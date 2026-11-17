// Package walwatch следит за слотами репликации Debezium в базах-источниках.
//
// Слот держит WAL всего сервера Postgres, пока Debezium не подтвердит позицию. Если
// Connect остановлен, база-источник «тихая» или слот брошен, журнал растёт, пока не
// заполнит диск. Наблюдатель предупреждает об этом заранее; слоты он не удаляет —
// это решение о потере данных, его принимает человек или лимит max_slot_wal_keep_size.
package walwatch

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"analyticDataCenter/analytics-data-center/internal/services/debezium"
)

const (
	checkTimeout = 30 * time.Second
	bytesInMB    = 1024 * 1024
)

// Slot — логический слот репликации.
type Slot struct {
	Name string
	// Active — к слоту подключён Debezium.
	Active bool
	// WALStatus (Postgres 13+): reserved, extended — норма; unreserved — WAL слота
	// превысил max_slot_wal_keep_size, слот скоро будет потерян; lost — потерян.
	WALStatus string
	// RetainedBytes — сколько WAL держит слот; -1 — неизвестно (у потерянного слота).
	RetainedBytes int64
}

// SourceDB — запросы к одной базе-источнику.
type SourceDB interface {
	// Slots — логические слоты Debezium (slot_*) сервера, на котором находится источник.
	Slots(ctx context.Context) ([]Slot, error)
	// MaxSlotWALKeepSize — значение max_slot_wal_keep_size; "-1" — лимита нет.
	MaxSlotWALKeepSize(ctx context.Context) (string, error)
}

// Opener открывает доступ к источнику по имени из конфига.
type Opener func(ctx context.Context, source string) (SourceDB, error)

type Watcher struct {
	log       *loggerpkg.Logger
	sources   []string
	open      Opener
	interval  time.Duration
	warnBytes int64

	// limitChecked — источники, у которых уже проверен max_slot_wal_keep_size.
	// Доступ только из Check, а Check выполняется в одной горутине.
	limitChecked map[string]bool

	ctx       context.Context
	cancel    context.CancelFunc
	stopped   chan struct{}
	startOnce sync.Once
	started   bool
	mu        sync.Mutex
}

// New: sources — имена источников из конфига; warnBytes — порог предупреждения
// об удерживаемом WAL.
func New(log *loggerpkg.Logger, sources []string, open Opener, interval time.Duration, warnBytes int64) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		log:          log,
		sources:      sources,
		open:         open,
		interval:     interval,
		warnBytes:    warnBytes,
		limitChecked: make(map[string]bool),
		ctx:          ctx,
		cancel:       cancel,
		stopped:      make(chan struct{}),
	}
}

// Start запускает проверки: первая — сразу, дальше раз в interval.
func (w *Watcher) Start() {
	w.startOnce.Do(func() {
		w.mu.Lock()
		w.started = true
		w.mu.Unlock()
		go w.run()
	})
}

// Stop прерывает текущую проверку и ждёт остановки. Безопасен без Start и на nil
// (наблюдатель выключен в конфиге).
func (w *Watcher) Stop() {
	if w == nil {
		return
	}
	w.cancel()
	w.mu.Lock()
	started := w.started
	w.mu.Unlock()
	if started {
		<-w.stopped
	}
}

func (w *Watcher) run() {
	defer close(w.stopped)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(w.ctx, checkTimeout)
		w.Check(ctx)
		cancel()
		select {
		case <-ticker.C:
		case <-w.ctx.Done():
			return
		}
	}
}

// Check проверяет слоты всех источников и пишет итог в лог.
func (w *Watcher) Check(ctx context.Context) {
	slots := make(map[string]Slot)
	queried := make(map[string]bool)
	for _, source := range w.sources {
		db, err := w.open(ctx, source)
		if err != nil {
			w.log.WarnMsg(loggerpkg.MsgWALCheckFailed, slog.String("source", source), slog.String("error", err.Error()))
			continue
		}
		w.checkSlotLimit(ctx, source, db)
		list, err := db.Slots(ctx)
		if err != nil {
			w.log.WarnMsg(loggerpkg.MsgWALCheckFailed, slog.String("source", source), slog.String("error", err.Error()))
			continue
		}
		queried[source] = true
		// Источники одного сервера видят одни и те же слоты — объединяем по имени.
		for _, s := range list {
			slots[s.Name] = s
		}
	}
	for _, f := range evaluate(w.sources, queried, slots, w.warnBytes) {
		w.report(f)
	}
	if len(queried) > 0 {
		w.logSummary(slots)
	}
}

// checkSlotLimit предупреждает, если у сервера источника нет лимита WAL для слотов.
// Проверяется один раз на источник: на Postgres до 13 настройки нет, и ошибка
// повторялась бы на каждой проверке.
func (w *Watcher) checkSlotLimit(ctx context.Context, source string, db SourceDB) {
	if w.limitChecked[source] {
		return
	}
	w.limitChecked[source] = true
	value, err := db.MaxSlotWALKeepSize(ctx)
	switch {
	case err != nil:
		w.log.WarnMsg(loggerpkg.MsgWALSlotLimitUnknown, slog.String("source", source), slog.String("error", err.Error()))
	case value == "-1":
		w.log.WarnMsg(loggerpkg.MsgWALNoSlotLimit, slog.String("source", source))
	}
}

type findingKind int

const (
	findingMissing    findingKind = iota + 1 // слота источника нет
	findingLost                              // слот потерян
	findingUnreserved                        // слот вот-вот будет потерян
	findingInactive                          // Debezium не читает слот
	findingRetained                          // слот держит WAL больше порога
	findingOrphan                            // слот без источника в конфиге
)

type finding struct {
	kind   findingKind
	source string // пусто у брошенного слота
	slot   Slot
}

// evaluate решает, о чём предупредить. Отсутствие слота сообщается только для
// источника, который удалось опросить: при ошибке запроса слот мог просто не прочитаться.
func evaluate(sources []string, queried map[string]bool, slots map[string]Slot, warnBytes int64) []finding {
	var out []finding
	configured := make(map[string]bool, len(sources))
	for _, source := range sources {
		name := debezium.SlotName(source)
		configured[name] = true
		slot, ok := slots[name]
		if !ok {
			if queried[source] {
				out = append(out, finding{kind: findingMissing, source: source, slot: Slot{Name: name}})
			}
			continue
		}
		out = append(out, slotFindings(source, slot, warnBytes)...)
	}

	var orphans []string
	for name := range slots {
		if !configured[name] {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	for _, name := range orphans {
		out = append(out, finding{kind: findingOrphan, slot: slots[name]})
	}
	return out
}

func slotFindings(source string, s Slot, warnBytes int64) []finding {
	if s.WALStatus == "lost" {
		// Потерянный слот неактивен и WAL не держит — остальные проверки бессмысленны.
		return []finding{{kind: findingLost, source: source, slot: s}}
	}
	var out []finding
	if s.WALStatus == "unreserved" {
		out = append(out, finding{kind: findingUnreserved, source: source, slot: s})
	}
	if !s.Active {
		out = append(out, finding{kind: findingInactive, source: source, slot: s})
	}
	if s.RetainedBytes > warnBytes {
		out = append(out, finding{kind: findingRetained, source: source, slot: s})
	}
	return out
}

func (w *Watcher) report(f finding) {
	attrs := []any{slog.String("slot", f.slot.Name)}
	if f.source != "" {
		attrs = append(attrs, slog.String("source", f.source))
	}
	if f.slot.RetainedBytes >= 0 && f.kind != findingMissing {
		attrs = append(attrs, slog.Int64("retained_mb", f.slot.RetainedBytes/bytesInMB))
	}
	switch f.kind {
	case findingMissing:
		w.log.WarnMsg(loggerpkg.MsgWALSlotMissing, attrs...)
	case findingLost:
		w.log.ErrorMsg(loggerpkg.MsgWALSlotLost, attrs...)
	case findingUnreserved:
		w.log.ErrorMsg(loggerpkg.MsgWALSlotUnreserved, attrs...)
	case findingInactive:
		w.log.WarnMsg(loggerpkg.MsgWALSlotInactive, attrs...)
	case findingRetained:
		w.log.WarnMsg(loggerpkg.MsgWALSlotRetained, append(attrs, slog.Int64("warn_mb", w.warnBytes/bytesInMB))...)
	case findingOrphan:
		w.log.WarnMsg(loggerpkg.MsgWALSlotOrphan, append(attrs, slog.Bool("active", f.slot.Active))...)
	}
}

func (w *Watcher) logSummary(slots map[string]Slot) {
	var maxRetained int64
	for _, s := range slots {
		maxRetained = max(maxRetained, s.RetainedBytes)
	}
	w.log.InfoMsg(loggerpkg.MsgWALSummary,
		slog.Int("slots", len(slots)),
		slog.Int64("max_retained_mb", maxRetained/bytesInMB),
	)
}

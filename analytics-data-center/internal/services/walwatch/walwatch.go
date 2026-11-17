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
	"net"
	"net/url"
	"sort"
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"analyticDataCenter/analytics-data-center/internal/services/debezium"
)

const (
	// sourceCheckTimeout — сколько ждать ответа одного источника. Таймаут у каждого
	// источника свой: завис один сервер — остальные всё равно проверяются.
	sourceCheckTimeout = 10 * time.Second
	bytesInMB          = 1024 * 1024
)

// Source — источник из конфига и сервер Postgres, на котором он находится.
type Source struct {
	Name string
	// Server — «host:port» сервера (ServerFromConnString). Имена слотов уникальны только
	// в пределах сервера, поэтому слоты разных серверов оцениваются отдельно. Пусто —
	// сервер неизвестен, источник считается отдельным сервером.
	Server string
}

// ServerFromConnString — «host:port» сервера Postgres из строки подключения вида
// postgresql://...; пусто, если строку не удалось разобрать. Один сервер, записанный
// под разными адресами, даст повторные предупреждения, но не пропущенные.
func ServerFromConnString(conn string) string {
	u, err := url.Parse(conn)
	if err != nil || u.Hostname() == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return ""
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	return net.JoinHostPort(u.Hostname(), port)
}

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
	log           *loggerpkg.Logger
	sources       []Source
	open          Opener
	interval      time.Duration
	warnBytes     int64
	sourceTimeout time.Duration

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

// New: warnBytes — порог предупреждения об удерживаемом WAL.
func New(log *loggerpkg.Logger, sources []Source, open Opener, interval time.Duration, warnBytes int64) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		log:           log,
		sources:       sources,
		open:          open,
		interval:      interval,
		warnBytes:     warnBytes,
		sourceTimeout: sourceCheckTimeout,
		limitChecked:  make(map[string]bool),
		ctx:           ctx,
		cancel:        cancel,
		stopped:       make(chan struct{}),
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
		// Общий контекст нужен только для остановки; время ограничено у каждого источника.
		w.Check(w.ctx)
		select {
		case <-ticker.C:
		case <-w.ctx.Done():
			return
		}
	}
}

// Check проверяет слоты всех источников и пишет итог в лог.
func (w *Watcher) Check(ctx context.Context) {
	observed := make(map[string]map[string]Slot)
	for _, src := range w.sources {
		if slots, ok := w.checkSource(ctx, src); ok {
			observed[src.Name] = slots
		}
	}
	for _, f := range evaluate(w.sources, observed, w.warnBytes) {
		w.report(f)
	}
	if len(observed) > 0 {
		w.logSummary(observed)
	}
}

// checkSource опрашивает один источник со своим таймаутом. ok=false — ответа нет,
// и об источнике в этой проверке ничего не известно.
func (w *Watcher) checkSource(ctx context.Context, src Source) (map[string]Slot, bool) {
	ctx, cancel := context.WithTimeout(ctx, w.sourceTimeout)
	defer cancel()
	db, err := w.open(ctx, src.Name)
	if err != nil {
		w.log.WarnMsg(loggerpkg.MsgWALCheckFailed, slog.String("source", src.Name), slog.String("error", err.Error()))
		return nil, false
	}
	w.checkSlotLimit(ctx, src.Name, db)
	list, err := db.Slots(ctx)
	if err != nil {
		w.log.WarnMsg(loggerpkg.MsgWALCheckFailed, slog.String("source", src.Name), slog.String("error", err.Error()))
		return nil, false
	}
	slots := make(map[string]Slot, len(list))
	for _, s := range list {
		slots[s.Name] = s
	}
	return slots, true
}

// checkSlotLimit предупреждает, если у сервера источника нет лимита WAL для слотов.
// Проверяется один раз на источник: на Postgres до 13 настройки нет, и ошибка
// повторялась бы на каждой проверке. Таймаут — не ответ сервера: тогда повторим.
func (w *Watcher) checkSlotLimit(ctx context.Context, source string, db SourceDB) {
	if w.limitChecked[source] {
		return
	}
	value, err := db.MaxSlotWALKeepSize(ctx)
	if err != nil && ctx.Err() != nil {
		return
	}
	w.limitChecked[source] = true
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
	findingOrphan                            // слот без источника этого сервера в конфиге
)

type finding struct {
	kind   findingKind
	source string // пусто у брошенного слота
	server string
	slot   Slot
}

// serverSlots — слоты одного сервера, собранные из ответов его источников.
type serverSlots struct {
	server string
	slots  map[string]Slot
	own    map[string]bool // слоты источников этого сервера
}

// evaluate решает, о чём предупредить. observed — слоты сервера каждого источника,
// который удалось опросить.
//
// Имя слота уникально только в пределах сервера, поэтому свой слот источника ищется
// только в ответе его сервера, а брошенные слоты — по каждому серверу отдельно:
// одноимённый слот другого сервера не скрывает потерянный или отсутствующий.
func evaluate(sources []Source, observed map[string]map[string]Slot, warnBytes int64) []finding {
	var out []finding
	servers := make(map[string]*serverSlots)
	// Чей слот — решает конфиг, а не успех запроса: слот источника, который сейчас не
	// ответил, виден в ответе соседа по серверу и не должен выглядеть брошенным.
	for _, src := range sources {
		key := serverKey(src)
		srv, ok := servers[key]
		if !ok {
			srv = &serverSlots{server: src.Server, slots: make(map[string]Slot), own: make(map[string]bool)}
			servers[key] = srv
		}
		srv.own[debezium.SlotName(src.Name)] = true
	}
	// Сервер источника неизвестен (строка подключения не URL) — он может оказаться любым
	// из опрошенных, поэтому его слот нигде не считается брошенным.
	for _, src := range sources {
		if src.Server != "" {
			continue
		}
		for _, srv := range servers {
			srv.own[debezium.SlotName(src.Name)] = true
		}
	}

	for _, src := range sources {
		slots, ok := observed[src.Name]
		if !ok {
			continue // ответа нет — об источнике ничего не известно, «не найден» не сообщаем
		}
		name := debezium.SlotName(src.Name)
		if slot, found := slots[name]; found {
			out = append(out, slotFindings(src, slot, warnBytes)...)
		} else {
			out = append(out, finding{kind: findingMissing, source: src.Name, server: src.Server, slot: Slot{Name: name}})
		}
		// Источники одного сервера видят одни и те же слоты — объединяем их ответы.
		srv := servers[serverKey(src)]
		for n, s := range slots {
			srv.slots[n] = s
		}
	}

	keys := make([]string, 0, len(servers))
	for key := range servers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		srv := servers[key]
		var orphans []string
		for name := range srv.slots {
			if !srv.own[name] {
				orphans = append(orphans, name)
			}
		}
		sort.Strings(orphans)
		for _, name := range orphans {
			out = append(out, finding{kind: findingOrphan, server: srv.server, slot: srv.slots[name]})
		}
	}
	return out
}

// serverKey группирует источники по серверу; источник с неизвестным сервером — отдельная группа.
func serverKey(src Source) string {
	if src.Server == "" {
		return "source:" + src.Name
	}
	return src.Server
}

func slotFindings(src Source, s Slot, warnBytes int64) []finding {
	if s.WALStatus == "lost" {
		// Потерянный слот неактивен и WAL не держит — остальные проверки бессмысленны.
		return []finding{{kind: findingLost, source: src.Name, server: src.Server, slot: s}}
	}
	var out []finding
	if s.WALStatus == "unreserved" {
		out = append(out, finding{kind: findingUnreserved, source: src.Name, server: src.Server, slot: s})
	}
	if !s.Active {
		out = append(out, finding{kind: findingInactive, source: src.Name, server: src.Server, slot: s})
	}
	if s.RetainedBytes > warnBytes {
		out = append(out, finding{kind: findingRetained, source: src.Name, server: src.Server, slot: s})
	}
	return out
}

func (w *Watcher) report(f finding) {
	attrs := []any{slog.String("slot", f.slot.Name)}
	if f.source != "" {
		attrs = append(attrs, slog.String("source", f.source))
	}
	if f.server != "" {
		attrs = append(attrs, slog.String("server", f.server))
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

// logSummary — сколько слотов на опрошенных серверах и максимум удерживаемого WAL.
func (w *Watcher) logSummary(observed map[string]map[string]Slot) {
	unique := make(map[string]Slot)
	for _, src := range w.sources {
		slots, ok := observed[src.Name]
		if !ok {
			continue
		}
		for name, s := range slots {
			unique[serverKey(src)+"/"+name] = s
		}
	}
	var maxRetained int64
	for _, s := range unique {
		maxRetained = max(maxRetained, s.RetainedBytes)
	}
	w.log.InfoMsg(loggerpkg.MsgWALSummary,
		slog.Int("slots", len(unique)),
		slog.Int64("max_retained_mb", maxRetained/bytesInMB),
	)
}

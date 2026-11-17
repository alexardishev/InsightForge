package walwatch

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/stretchr/testify/require"
)

const mb = 1024 * 1024

func slotsOf(slots ...Slot) map[string]Slot {
	m := make(map[string]Slot, len(slots))
	for _, s := range slots {
		m[s.Name] = s
	}
	return m
}

func TestEvaluate(t *testing.T) {
	db1 := []Source{{Name: "db1", Server: "a:5432"}}
	healthy := Slot{Name: "slot_db1", Active: true, WALStatus: "reserved", RetainedBytes: 10 * mb}
	for _, tc := range []struct {
		name     string
		observed map[string]map[string]Slot // источник → слоты его сервера
		want     []finding
	}{
		{"healthy slot", map[string]map[string]Slot{"db1": slotsOf(healthy)}, nil},
		{
			"slot of a configured source not found",
			map[string]map[string]Slot{"db1": slotsOf()},
			[]finding{{kind: findingMissing, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1"}}},
		},
		{"source not queried is not reported as missing", map[string]map[string]Slot{}, nil},
		{
			"Connect does not read the slot",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", WALStatus: "reserved", RetainedBytes: mb})},
			[]finding{{kind: findingInactive, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", WALStatus: "reserved", RetainedBytes: mb}}},
		},
		{
			"retained WAL above the threshold",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", Active: true, WALStatus: "extended", RetainedBytes: 101 * mb})},
			[]finding{{kind: findingRetained, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", Active: true, WALStatus: "extended", RetainedBytes: 101 * mb}}},
		},
		{
			"retained WAL exactly at the threshold",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", Active: true, WALStatus: "reserved", RetainedBytes: 100 * mb})},
			nil,
		},
		{
			"inactive and above the threshold: both reported",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb})},
			[]finding{
				{kind: findingInactive, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb}},
				{kind: findingRetained, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb}},
			},
		},
		{
			"slot about to be invalidated",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", Active: true, WALStatus: "unreserved", RetainedBytes: 50 * mb})},
			[]finding{{kind: findingUnreserved, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", Active: true, WALStatus: "unreserved", RetainedBytes: 50 * mb}}},
		},
		{
			"lost slot is reported once, without inactive and retained",
			map[string]map[string]Slot{"db1": slotsOf(Slot{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1})},
			[]finding{{kind: findingLost, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1}}},
		},
		{
			"slot of a source that is not in the config",
			map[string]map[string]Slot{"db1": slotsOf(healthy, Slot{Name: "slot_old", WALStatus: "extended", RetainedBytes: 300 * mb})},
			[]finding{{kind: findingOrphan, server: "a:5432", slot: Slot{Name: "slot_old", WALStatus: "extended", RetainedBytes: 300 * mb}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, evaluate(db1, tc.observed, 100*mb))
		})
	}
}

func TestEvaluateKeepsSlotsOfDifferentServersApart(t *testing.T) {
	// Имя слота уникально только в пределах сервера: на сервере B остался свой slot_db1.
	sources := []Source{{Name: "db1", Server: "a:5432"}, {Name: "db2", Server: "b:5432"}}
	leftover := Slot{Name: "slot_db1", Active: true, WALStatus: "reserved"}
	db2 := Slot{Name: "slot_db2", Active: true, WALStatus: "reserved"}

	t.Run("lost slot is not masked by a healthy slot with the same name on another server", func(t *testing.T) {
		lost := Slot{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1}
		got := evaluate(sources, map[string]map[string]Slot{
			"db1": slotsOf(lost),
			"db2": slotsOf(db2, leftover),
		}, 100*mb)

		require.Equal(t, []finding{
			{kind: findingLost, source: "db1", server: "a:5432", slot: lost},
			{kind: findingOrphan, server: "b:5432", slot: leftover},
		}, got)
	})

	t.Run("missing slot is not hidden by a slot with the same name on another server", func(t *testing.T) {
		got := evaluate(sources, map[string]map[string]Slot{
			"db1": slotsOf(),
			"db2": slotsOf(db2, leftover),
		}, 100*mb)

		require.Equal(t, []finding{
			{kind: findingMissing, source: "db1", server: "a:5432", slot: Slot{Name: "slot_db1"}},
			{kind: findingOrphan, server: "b:5432", slot: leftover},
		}, got)
	})
}

func TestServerFromConnString(t *testing.T) {
	require.Equal(t, "pg:5433", ServerFromConnString("postgresql://u:p@pg:5433/shop?sslmode=disable"))
	require.Equal(t, "pg:5432", ServerFromConnString("postgres://u:p@pg/shop"), "default Postgres port")
	require.Empty(t, ServerFromConnString("host=pg port=5432 dbname=shop"), "key=value DSN is not parsed")
	require.Empty(t, ServerFromConnString("://bad"))
}

// fakeSource — источник с заданными слотами и настройкой лимита. stall — сервер завис:
// запросы ждут, пока не истечёт контекст. Как драйвер Postgres, на истёкшем контексте
// запрос сразу возвращает ошибку.
type fakeSource struct {
	mu        sync.Mutex
	slots     []Slot
	slotsErr  error
	keepSize  string
	keepErr   error
	keepCalls int
	stall     bool
	checked   chan struct{}
}

func (s *fakeSource) Slots(ctx context.Context) ([]Slot, error) {
	if s.stall {
		<-ctx.Done()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.checked != nil {
		select {
		case s.checked <- struct{}{}:
		default:
		}
	}
	return s.slots, s.slotsErr
}

func (s *fakeSource) MaxSlotWALKeepSize(ctx context.Context) (string, error) {
	if s.stall {
		<-ctx.Done()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keepCalls++
	return s.keepSize, s.keepErr
}

// newTestWatcher: все источники — на одном сервере, если не задан servers.
func newTestWatcher(sources map[string]*fakeSource, order ...string) (*Watcher, *strings.Builder) {
	var list []Source
	for _, name := range order {
		list = append(list, Source{Name: name, Server: "pg:5432"})
	}
	return newTestWatcherOn(sources, list)
}

func newTestWatcherOn(sources map[string]*fakeSource, list []Source) (*Watcher, *strings.Builder) {
	out := &strings.Builder{}
	log := &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(out, nil))}
	open := func(_ context.Context, name string) (SourceDB, error) {
		src, ok := sources[name]
		if !ok {
			return nil, errors.New("source not configured")
		}
		return src, nil
	}
	return New(log, list, open, time.Hour, 100*mb), out
}

func TestCheckWarnsOnceAboutMissingSlotLimit(t *testing.T) {
	src := &fakeSource{keepSize: "-1", slots: []Slot{{Name: "slot_db1", Active: true, WALStatus: "reserved"}}}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")

	w.Check(context.Background())
	w.Check(context.Background())

	require.Equal(t, 1, strings.Count(out.String(), loggerpkg.MsgWALNoSlotLimit.RU))
	require.Equal(t, 1, src.keepCalls, "the limit is checked once per source")
}

func TestCheckDoesNotWarnWhenSlotLimitIsSet(t *testing.T) {
	src := &fakeSource{keepSize: "10GB", slots: []Slot{{Name: "slot_db1", Active: true, WALStatus: "reserved"}}}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")

	w.Check(context.Background())

	require.NotContains(t, out.String(), loggerpkg.MsgWALNoSlotLimit.RU)
}

func TestCheckReportsLimitQueryErrorOnce(t *testing.T) {
	// Postgres до 13 не знает max_slot_wal_keep_size: сообщить один раз, а не раз в минуту.
	src := &fakeSource{keepErr: errors.New(`unrecognized configuration parameter "max_slot_wal_keep_size"`)}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")

	w.Check(context.Background())
	w.Check(context.Background())

	require.Equal(t, 1, src.keepCalls)
	require.Equal(t, 1, strings.Count(out.String(), "max_slot_wal_keep_size"))
}

func TestCheckQueryFailureIsNotReportedAsMissingSlot(t *testing.T) {
	src := &fakeSource{keepSize: "10GB", slotsErr: errors.New("permission denied")}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")

	w.Check(context.Background())

	require.Contains(t, out.String(), loggerpkg.MsgWALCheckFailed.RU)
	require.NotContains(t, out.String(), loggerpkg.MsgWALSlotMissing.RU)
}

func TestCheckContinuesAfterUnavailableSource(t *testing.T) {
	db2 := &fakeSource{keepSize: "10GB", slots: []Slot{{Name: "slot_db2", WALStatus: "reserved"}}}
	w, out := newTestWatcher(map[string]*fakeSource{"db2": db2}, "db1", "db2")

	w.Check(context.Background())

	require.Contains(t, out.String(), loggerpkg.MsgWALCheckFailed.RU)
	require.Contains(t, out.String(), loggerpkg.MsgWALSlotInactive.RU, "db2 is still checked")
	require.NotContains(t, out.String(), loggerpkg.MsgWALSlotMissing.RU)
}

func TestCheckGivesEachSourceItsOwnTimeout(t *testing.T) {
	// Сервер db1 завис: он не должен съесть время проверки остальных источников.
	stalled := &fakeSource{stall: true}
	db2 := &fakeSource{keepSize: "10GB", slots: []Slot{{Name: "slot_db2", WALStatus: "lost", RetainedBytes: -1}}}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": stalled, "db2": db2}, "db1", "db2")
	w.sourceTimeout = 50 * time.Millisecond

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Check(context.Background())
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a stalled source must time out on its own")
	}

	logs := out.String()
	require.Contains(t, logs, loggerpkg.MsgWALCheckFailed.RU)
	require.Contains(t, logs, loggerpkg.MsgWALSlotLost.RU, "db2 is checked with its own time budget")
}

func TestCheckRetriesSlotLimitAfterTimeout(t *testing.T) {
	// Таймаут — не ответ сервера: проверку лимита нужно повторить на следующей проверке.
	src := &fakeSource{stall: true}
	w, out := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")
	w.sourceTimeout = 20 * time.Millisecond

	w.Check(context.Background())
	src.stall = false
	src.keepSize = "-1"
	w.Check(context.Background())

	require.Contains(t, out.String(), loggerpkg.MsgWALNoSlotLimit.RU)
	require.NotContains(t, out.String(), loggerpkg.MsgWALSlotLimitUnknown.RU)
}

func TestCheckMergesSlotsSeenFromSeveralSourcesOfOneServer(t *testing.T) {
	// Источники на одном сервере видят одни и те же слоты: брошенный слот — одно предупреждение.
	serverSlots := []Slot{
		{Name: "slot_db1", Active: true, WALStatus: "reserved"},
		{Name: "slot_db2", Active: true, WALStatus: "reserved"},
		{Name: "slot_old", WALStatus: "extended", RetainedBytes: 7 * mb},
	}
	w, out := newTestWatcher(map[string]*fakeSource{
		"db1": {keepSize: "10GB", slots: serverSlots},
		"db2": {keepSize: "10GB", slots: serverSlots},
	}, "db1", "db2")

	w.Check(context.Background())

	logs := out.String()
	require.Equal(t, 1, strings.Count(logs, loggerpkg.MsgWALSlotOrphan.RU))
	require.Contains(t, logs, "slot=slot_old")
	require.NotContains(t, logs, loggerpkg.MsgWALSlotMissing.RU)
	require.Contains(t, logs, loggerpkg.MsgWALSummary.RU)
	require.Contains(t, logs, "slots=3")
	require.Contains(t, logs, "max_retained_mb=7")
}

func TestCheckReportsLostSlotDespiteSameNameOnAnotherServer(t *testing.T) {
	w, out := newTestWatcherOn(map[string]*fakeSource{
		"db1": {keepSize: "10GB", slots: []Slot{{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1}}},
		"db2": {keepSize: "10GB", slots: []Slot{
			{Name: "slot_db2", Active: true, WALStatus: "reserved"},
			{Name: "slot_db1", Active: true, WALStatus: "reserved"},
		}},
	}, []Source{{Name: "db1", Server: "a:5432"}, {Name: "db2", Server: "b:5432"}})

	w.Check(context.Background())

	logs := out.String()
	require.Contains(t, logs, loggerpkg.MsgWALSlotLost.RU)
	require.Contains(t, logs, loggerpkg.MsgWALSlotOrphan.RU)
	require.Contains(t, logs, "server=b:5432")
}

func TestWatcherChecksRightAwayAndStops(t *testing.T) {
	src := &fakeSource{keepSize: "10GB", checked: make(chan struct{}, 1)}
	w, _ := newTestWatcher(map[string]*fakeSource{"db1": src}, "db1")

	w.Start()
	select {
	case <-src.checked:
	case <-time.After(2 * time.Second):
		t.Fatal("the first check must run right after Start, not after the interval")
	}

	stopped := make(chan struct{})
	go func() {
		w.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must not wait for the next interval")
	}
}

func TestSlotNamePatternMatchesOnlyDebeziumSlots(t *testing.T) {
	require.Equal(t, `slot\_%`, slotNamePattern(), "_ is escaped: in LIKE it matches any character")
}

func TestStopIsSafeWithoutStartAndOnNil(t *testing.T) {
	w, _ := newTestWatcher(nil)
	w.Stop()

	var disabled *Watcher // наблюдатель выключен в конфиге
	disabled.Stop()
}

func TestEvaluateUnavailableSourceStillOwnsItsSlot(t *testing.T) {
	// db1 и db2 на одном сервере; db1 не ответил, а ответ db2 содержит все слоты сервера.
	sources := []Source{{Name: "db1", Server: "a:5432"}, {Name: "db2", Server: "a:5432"}}
	got := evaluate(sources, map[string]map[string]Slot{
		"db2": slotsOf(
			Slot{Name: "slot_db1", Active: true, WALStatus: "reserved"},
			Slot{Name: "slot_db2", Active: true, WALStatus: "reserved"},
		),
	}, 100*mb)

	require.Empty(t, got, "slot_db1 belongs to a configured source: it is not orphaned just because db1 did not answer")
}

func TestCheckUnavailableSourceSlotIsNotReportedAsOrphan(t *testing.T) {
	w, out := newTestWatcher(map[string]*fakeSource{
		"db2": {keepSize: "10GB", slots: []Slot{
			{Name: "slot_db1", Active: true, WALStatus: "reserved"},
			{Name: "slot_db2", Active: true, WALStatus: "reserved"},
		}},
	}, "db1", "db2") // db1 недоступен, оба источника на одном сервере

	w.Check(context.Background())

	logs := out.String()
	require.Contains(t, logs, loggerpkg.MsgWALCheckFailed.RU)
	require.NotContains(t, logs, loggerpkg.MsgWALSlotOrphan.RU)
	require.NotContains(t, logs, loggerpkg.MsgWALSlotMissing.RU)
}

func TestEvaluateSourceWithUnknownServerOwnsItsSlotEverywhere(t *testing.T) {
	// Сервер db1 неизвестен (строка подключения не URL): его слот в ответе db2 — не брошенный.
	sources := []Source{{Name: "db1"}, {Name: "db2", Server: "a:5432"}}
	got := evaluate(sources, map[string]map[string]Slot{
		"db2": slotsOf(
			Slot{Name: "slot_db1", Active: true, WALStatus: "reserved"},
			Slot{Name: "slot_db2", Active: true, WALStatus: "reserved"},
		),
	}, 100*mb)

	require.Empty(t, got)
}

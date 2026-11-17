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

func TestEvaluate(t *testing.T) {
	healthy := Slot{Name: "slot_db1", Active: true, WALStatus: "reserved", RetainedBytes: 10 * mb}
	for _, tc := range []struct {
		name    string
		queried map[string]bool
		slots   []Slot
		want    []finding
	}{
		{"healthy slot", map[string]bool{"db1": true}, []Slot{healthy}, nil},
		{
			"slot of a configured source not found",
			map[string]bool{"db1": true}, nil,
			[]finding{{kind: findingMissing, source: "db1", slot: Slot{Name: "slot_db1"}}},
		},
		{"source not queried is not reported as missing", map[string]bool{}, nil, nil},
		{
			"Connect does not read the slot",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", WALStatus: "reserved", RetainedBytes: mb}},
			[]finding{{kind: findingInactive, source: "db1", slot: Slot{Name: "slot_db1", WALStatus: "reserved", RetainedBytes: mb}}},
		},
		{
			"retained WAL above the threshold",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", Active: true, WALStatus: "extended", RetainedBytes: 101 * mb}},
			[]finding{{kind: findingRetained, source: "db1", slot: Slot{Name: "slot_db1", Active: true, WALStatus: "extended", RetainedBytes: 101 * mb}}},
		},
		{
			"retained WAL exactly at the threshold",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", Active: true, WALStatus: "reserved", RetainedBytes: 100 * mb}},
			nil,
		},
		{
			"inactive and above the threshold: both reported",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb}},
			[]finding{
				{kind: findingInactive, source: "db1", slot: Slot{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb}},
				{kind: findingRetained, source: "db1", slot: Slot{Name: "slot_db1", WALStatus: "extended", RetainedBytes: 200 * mb}},
			},
		},
		{
			"slot about to be invalidated",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", Active: true, WALStatus: "unreserved", RetainedBytes: 50 * mb}},
			[]finding{{kind: findingUnreserved, source: "db1", slot: Slot{Name: "slot_db1", Active: true, WALStatus: "unreserved", RetainedBytes: 50 * mb}}},
		},
		{
			"lost slot is reported once, without inactive and retained",
			map[string]bool{"db1": true}, []Slot{{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1}},
			[]finding{{kind: findingLost, source: "db1", slot: Slot{Name: "slot_db1", WALStatus: "lost", RetainedBytes: -1}}},
		},
		{
			"slot of a source that is not in the config",
			map[string]bool{"db1": true}, []Slot{healthy, {Name: "slot_old", WALStatus: "extended", RetainedBytes: 300 * mb}},
			[]finding{{kind: findingOrphan, slot: Slot{Name: "slot_old", WALStatus: "extended", RetainedBytes: 300 * mb}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slots := make(map[string]Slot)
			for _, s := range tc.slots {
				slots[s.Name] = s
			}
			require.Equal(t, tc.want, evaluate([]string{"db1"}, tc.queried, slots, 100*mb))
		})
	}
}

// fakeSource — источник с заданными слотами и настройкой лимита.
type fakeSource struct {
	mu        sync.Mutex
	slots     []Slot
	slotsErr  error
	keepSize  string
	keepErr   error
	keepCalls int
	checked   chan struct{}
}

func (s *fakeSource) Slots(context.Context) ([]Slot, error) {
	if s.checked != nil {
		select {
		case s.checked <- struct{}{}:
		default:
		}
	}
	return s.slots, s.slotsErr
}

func (s *fakeSource) MaxSlotWALKeepSize(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keepCalls++
	return s.keepSize, s.keepErr
}

func newTestWatcher(sources map[string]*fakeSource, order ...string) (*Watcher, *strings.Builder) {
	out := &strings.Builder{}
	log := &loggerpkg.Logger{Logger: slog.New(slog.NewTextHandler(out, nil))}
	open := func(_ context.Context, name string) (SourceDB, error) {
		src, ok := sources[name]
		if !ok {
			return nil, errors.New("source not configured")
		}
		return src, nil
	}
	return New(log, order, open, time.Hour, 100*mb), out
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

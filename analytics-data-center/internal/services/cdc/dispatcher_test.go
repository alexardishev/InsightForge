package cdc

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"github.com/stretchr/testify/require"
	"testing"
)

type mockHandler struct {
	called bool
	event  models.CDCEvent
}

func (m *mockHandler) EventPreprocessing(evt models.CDCEvent) {
	m.called = true
	m.event = evt
}

func TestDispatch_ValidJSON(t *testing.T) {
	handler := &mockHandler{}
	data := []byte(`{"before":null,"after":{"id":1},"source":{"db":"test","schema":"public","table":"users"},"op":"c","transaction":null,"ts_ms":123}`)

	Dispatch(data, loggerpkg.New("test", "ru"), handler)

	require.True(t, handler.called, "handler should be called")
	require.Equal(t, "test", handler.event.Data.Source.DB)
	require.Equal(t, "users", handler.event.Data.Source.Table)
	require.Equal(t, "c", handler.event.Data.Op)
	require.Equal(t, float64(1), handler.event.Data.After["id"])
}

func TestDispatch_InvalidJSON(t *testing.T) {
	handler := &mockHandler{}
	data := []byte("{invalid json}")

	Dispatch(data, loggerpkg.New("test", "ru"), handler)

	require.False(t, handler.called, "handler should not be called on invalid JSON")
}

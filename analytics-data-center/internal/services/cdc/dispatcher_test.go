package cdc

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type ctxKey struct{}

type mockHandler struct {
	called bool
	event  models.CDCEvent
	ctx    context.Context
	err    error
}

func (m *mockHandler) EventPreprocessing(ctx context.Context, evt models.CDCEvent) error {
	m.called = true
	m.event = evt
	m.ctx = ctx
	return m.err
}

func TestDispatch_ValidJSON(t *testing.T) {
	handler := &mockHandler{}
	data := []byte(`{"before":null,"after":{"id":1},"source":{"db":"test","schema":"public","table":"users"},"op":"c","transaction":null,"ts_ms":123}`)
	ctx := context.WithValue(context.Background(), ctxKey{}, "listener")

	require.NoError(t, Dispatch(ctx, data, loggerpkg.New("test", "ru"), handler))

	require.True(t, handler.called, "handler should be called")
	require.Equal(t, "listener", handler.ctx.Value(ctxKey{}), "listener context reaches the handler")
	require.Equal(t, "test", handler.event.Data.Source.DB)
	require.Equal(t, "public", handler.event.Data.Source.Schema)
	require.Equal(t, "users", handler.event.Data.Source.Table)
	require.Equal(t, "c", handler.event.Data.Op)
}

func TestDispatch_InvalidJSONIsPermanent(t *testing.T) {
	handler := &mockHandler{}

	err := Dispatch(context.Background(), []byte("{invalid json}"), loggerpkg.New("test", "ru"), handler)
	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.True(t, IsPermanent(err), "retrying a malformed message cannot succeed")
	require.Contains(t, err.Error(), "Dispatch: JSON parse error:")
	require.False(t, handler.called, "handler should not be called on invalid JSON")
}

func TestDispatch_TombstoneIsSkipped(t *testing.T) {
	for _, value := range [][]byte{nil, {}} {
		handler := &mockHandler{}
		require.NoError(t, Dispatch(context.Background(), value, loggerpkg.New("test", "ru"), handler))
		require.False(t, handler.called)
	}
}

func TestDispatch_PreservesProcessingErrorClass(t *testing.T) {
	cause := errors.New("DWH unavailable")
	for _, tc := range []struct {
		name      string
		err       error
		permanent bool
	}{
		{"transient", cause, false},
		{"permanent", Permanent(cause), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := &mockHandler{err: tc.err}
			err := Dispatch(context.Background(), []byte(`{"op":"u"}`), loggerpkg.New("test", "en"), handler)
			require.ErrorIs(t, err, cause)
			require.Equal(t, tc.permanent, IsPermanent(err))
			require.Contains(t, err.Error(), "Dispatch: event processing failed:")
		})
	}
}

package cdc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermanent(t *testing.T) {
	cause := errors.New("view deleted")

	require.NoError(t, Permanent(nil))
	require.False(t, IsPermanent(cause))
	require.False(t, IsPermanent(nil))

	err := Permanent(cause)
	require.True(t, IsPermanent(err))
	require.ErrorIs(t, err, cause, "cause stays reachable")
	require.Equal(t, cause.Error(), err.Error())

	wrapped := fmt.Errorf("Dispatch: event processing failed: %w", err)
	require.True(t, IsPermanent(wrapped), "mark survives wrapping")
}

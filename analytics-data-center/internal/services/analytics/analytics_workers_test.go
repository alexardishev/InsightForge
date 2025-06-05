package serviceanalytics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateWorkerCount(t *testing.T) {
	svc := &AnalyticsDataCenterService{log: getTestLogger()}

	tests := []struct {
		name   string
		input  int64
		expect int64
	}{
		{"zero", 0, 1},
		{"boundary10k", 10000, 1},
		{"over10k", 10001, 2},
		{"over100k", 100001, 4},
		{"over200k", 200001, 6},
		{"over500k", 500001, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateWorkerCount(tt.input)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEventIdentifier(t *testing.T) {
	svc := &AnalyticsDataCenterService{log: getTestLogger()}

	tests := []struct {
		input    string
		expected string
	}{
		{"c", "c"},
		{"u", "u"},
		{"d", "d"},
		{"r", "r"},
		{"x", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := svc.eventIdentifier(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

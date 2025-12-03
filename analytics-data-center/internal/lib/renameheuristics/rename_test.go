package renameheuristics

import (
	"context"
	"testing"
)

// Проверяем, что переименование не предлагается, если типы колонок не совпадают
// (например, текстовая колонка не может внезапно стать числовой).
func TestDetectRenameCandidate_TypeMismatch(t *testing.T) {
	cfg := DetectorConfig{
		ColumnTypes: map[string]string{
			"name":  "text",
			"title": "integer",
		},
		OldCandidates: []string{"name"},
		NewCandidates: []string{"title"},
	}

	candidates, err := DetectRenameCandidate(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) > 0 {
		t.Fatalf("expected no rename candidate due to type mismatch, got: %+v", candidates)
	}
}

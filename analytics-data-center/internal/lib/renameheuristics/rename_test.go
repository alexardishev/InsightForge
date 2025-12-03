package renameheuristics

import (
	"context"
	"strings"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
)

// Проверяем, что переименование не предлагается, если типы колонок не совпадают
// (например, текстовая колонка не может внезапно стать числовой).
func TestDetectRenameCandidate_TypeMismatch(t *testing.T) {
	cfg := DetectorConfig{
		ActualDWHColumns: map[string]struct{}{strings.ToLower("name"): {}},
		ColumnTypes: map[string]string{
			"name":  "text",
			"title": "integer",
		},
		ExpectedColumns: []models.Column{{Name: "title", Type: "integer"}},
	}

	candidate, err := DetectRenameCandidate(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if candidate != nil {
		t.Fatalf("expected no rename candidate due to type mismatch, got: %+v", candidate)
	}
}

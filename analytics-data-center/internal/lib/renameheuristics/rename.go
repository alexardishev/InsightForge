package renameheuristics

import (
	"context"
	"log/slog"
	"strings"

	"analyticDataCenter/analytics-data-center/internal/domain/models"

	"github.com/adrg/strutil/metrics"
)

// RenameCandidate describes the most probable rename pair.
type RenameCandidate struct {
	OldName  string
	NewName  string
	Strategy string
}

// DetectorConfig aggregates inputs needed to detect rename patterns between схемой сервиса и DWH.
type DetectorConfig struct {
	ActualDWHColumns      map[string]struct{}
	BeforeEvent           map[string]interface{}
	AfterEvent            map[string]interface{}
	Database              string
	Schema                string
	Table                 string
	RenameHeuristicEnable bool
	ExpectedColumns       []models.Column
	Logger                *slog.Logger
}

// DetectRenameCandidate tries to infer a column rename using view-schema diff and CDC heuristics.
func DetectRenameCandidate(ctx context.Context, cfg DetectorConfig) (*RenameCandidate, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	expectedTypes := make(map[string]string)
	for _, col := range cfg.ExpectedColumns {
		if col.Name == "" {
			continue
		}
		expectedTypes[col.Name] = normalizeType(col.Type)
	}

	if len(expectedTypes) > 0 {
		missing, added := diffSets(cfg.ActualDWHColumns, expectedTypes)
		log.Info("", slog.Any("DWH", cfg.ActualDWHColumns))
		log.Info("", slog.Any("OLTP", expectedTypes))

		if candidate := pickCandidate(missing, added, expectedTypes, expectedTypes, "view-schema", log); candidate != nil {
			return candidate, nil
		}
	}

	if !cfg.RenameHeuristicEnable {
		return nil, nil
	}

	missing, added := diffEventColumns(cfg.ActualDWHColumns, cfg.AfterEvent)
	missing, added = filterByBefore(cfg.BeforeEvent, cfg.ExpectedColumns, missing, added)

	return pickCandidate(missing, added, expectedTypes, nil, "cdc-heuristic", log), nil
}

func pickCandidate(
	missing []string,
	added []string,
	expectedTypes map[string]string,
	newTypes map[string]string,
	strategy string,
	log *slog.Logger,
) *RenameCandidate {
	if len(missing) == 0 || len(added) == 0 {
		return nil
	}

	// В простых случаях с одной пропавшей колонкой позволяем более мягкое
	// сравнение, т.к. эвристика с Jaro-Winkler может не сработать на коротких
	// или сильно отличающихся названиях вроде name -> title. Даже если
	// добавленных колонок несколько, мы ищем «лучшего кандидата» среди них.
	minSimilarity := 0.82
	if len(missing) == 1 {
		minSimilarity = 0.6
	}

	type candidateScore struct {
		old   string
		new   string
		score float64
	}

	jw := metrics.NewJaroWinkler()
	var best *candidateScore

	for _, oldCol := range missing {
		expectedType := expectedTypes[oldCol]
		for _, newCol := range added {
			newType := normalizeType(newTypes[newCol])
			if expectedType != "" && newType != "" && expectedType != newType {
				// Тип изменился — вероятнее новая колонка, а не rename.
				log.Info("skip rename candidate due to type mismatch", slog.String("old", oldCol), slog.String("new", newCol), slog.String("expectedType", expectedType), slog.String("newType", newType))
				continue
			}

			similarity := jw.Compare(normalizeName(oldCol), normalizeName(newCol))
			if expectedType != "" && newType != "" && expectedType == newType {
				similarity += 0.1 // bonus for matching types
			}

			if similarity < minSimilarity { // too low semantic similarity
				continue
			}

			if best == nil || similarity > best.score {
				best = &candidateScore{old: oldCol, new: newCol, score: similarity}
			}
		}
	}

	if best == nil {
		return nil
	}

	return &RenameCandidate{OldName: best.old, NewName: best.new, Strategy: strategy}
}

func diffSets(actual map[string]struct{}, expected map[string]string) ([]string, []string) {
	var missing []string
	var added []string

	for col := range actual {
		if _, ok := expected[col]; !ok {
			missing = append(missing, col)
		}
	}

	for col := range expected {
		if _, ok := actual[col]; !ok {
			added = append(added, col)
		}
	}

	return missing, added
}

func diffEventColumns(actual map[string]struct{}, after map[string]interface{}) ([]string, []string) {
	var missing []string
	var added []string

	for col := range actual {
		if _, ok := after[col]; !ok {
			missing = append(missing, col)
		}
	}

	for col := range after {
		if _, ok := actual[col]; !ok {
			added = append(added, col)
		}
	}

	return missing, added
}

func filterByBefore(before map[string]interface{}, expected []models.Column, missing, added []string) ([]string, []string) {
	// Debezium в наших событиях не заполняет before, поэтому используем его, если он есть,
	// а в противном случае — список колонок из схемы (expected).
	beforeColumns := make(map[string]struct{})
	for col := range before {
		beforeColumns[col] = struct{}{}
	}

	if len(beforeColumns) == 0 {
		for _, col := range expected {
			if col.Name == "" {
				continue
			}
			beforeColumns[col.Name] = struct{}{}
		}
	}

	if len(beforeColumns) == 0 {
		return missing, added
	}

	var filteredMissing []string
	for _, m := range missing {
		if _, ok := beforeColumns[m]; ok {
			filteredMissing = append(filteredMissing, m)
		}
	}

	var filteredAdded []string
	for _, a := range added {
		if _, ok := beforeColumns[a]; !ok {
			filteredAdded = append(filteredAdded, a)
		}
	}

	return filteredMissing, filteredAdded
}

func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	return s
}

func normalizeType(t string) string {
	return strings.ToLower(strings.TrimSpace(t))
}

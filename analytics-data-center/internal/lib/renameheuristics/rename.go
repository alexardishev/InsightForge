package renameheuristics

import (
	"context"
	"log/slog"
	"strings"

	"github.com/adrg/strutil/metrics"
)

// RenameCandidate describes the most probable rename pair with similarity score.
type RenameCandidate struct {
	OldName string
	NewName string
	Score   float64
}

// DetectorConfig aggregates inputs needed to detect rename patterns between схемой сервиса и DWH.
type DetectorConfig struct {
	OldCandidates []string
	NewCandidates []string
	ColumnTypes   map[string]string
	Logger        *slog.Logger
}

// DetectRenameCandidate tries to infer rename pairs based on provided candidate sets.
func DetectRenameCandidate(ctx context.Context, cfg DetectorConfig) ([]RenameCandidate, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	allTypes := make(map[string]string)
	for name, t := range cfg.ColumnTypes {
		allTypes[strings.ToLower(name)] = normalizeType(t)
	}

	return pickCandidates(cfg.OldCandidates, cfg.NewCandidates, allTypes, log), nil
}

func pickCandidates(
	oldCandidates []string,
	newCandidates []string,
	types map[string]string,
	log *slog.Logger,
) []RenameCandidate {
	if len(oldCandidates) == 0 || len(newCandidates) == 0 {
		return nil
	}

	minSimilarity := 0.45
	jw := metrics.NewJaroWinkler()
	var results []RenameCandidate

	for _, oldCol := range oldCandidates {
		expectedType := types[strings.ToLower(oldCol)]
		for _, newCol := range newCandidates {
			newType := normalizeType(types[strings.ToLower(newCol)])
			if expectedType == "" && newType == "" {
				continue
			}
			if expectedType != "" && newType != "" && expectedType != newType {
				log.Debug("skip rename candidate due to type mismatch",
					slog.String("old", oldCol),
					slog.String("new", newCol),
					slog.String("expectedType", expectedType),
					slog.String("newType", newType))
				continue
			}

			similarity := jw.Compare(normalizeName(oldCol), normalizeName(newCol))
			if expectedType != "" && newType != "" && expectedType == newType {
				similarity += 0.1 // бонус за совпадение типов
			}

			if similarity < minSimilarity {
				continue
			}

			results = append(results, RenameCandidate{OldName: oldCol, NewName: newCol, Score: similarity})
		}
	}

	return results
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

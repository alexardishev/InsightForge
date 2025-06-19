package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"testing"
)

func TestGenerateQueryCreateTempTablePostgres(t *testing.T) {
	// Логгер, пишущий в консоль
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	view := &models.View{
		Name: "test_view",
		Sources: []models.Source{
			{
				Name: "db1",
				Schemas: []models.Schema{
					{
						Name: "public",
						Tables: []models.Table{
							{
								Name: "users",
								Columns: []models.Column{
									{Name: "id", Type: "INT"},
									{Name: "name", Type: "TEXT"},
									{Name: "name", Type: "TEXT"},
								},
							},
							{
								Name: "oop",
								Columns: []models.Column{
									{Name: "id", Type: "INT"},
									{Name: "name", Type: "TEXT"},
									{Name: "name", Type: "TEXT"},
									{Name: "name", Type: "TEXT"},
									{Name: "name", Type: "TEXT"},
								},
							},
						},
					},
					{
						Name: "google",
						Tables: []models.Table{
							{
								Name: "users",
								Columns: []models.Column{
									{Name: "id", Type: "INT"},
									{Name: "name", Type: "TEXT"},
								},
							},
							{
								Name: "oop",
								Columns: []models.Column{
									{Name: "id", Type: "INT"},
									{Name: "name", Type: "TEXT"},
								},
							},
						},
					},
				},
			},
			{
				Name: "db2",
				Schemas: []models.Schema{
					{
						Name: "analytics",
						Tables: []models.Table{
							{
								Name: "events",
								Columns: []models.Column{
									{Name: "event_id", Type: "UUID"},
									{Name: "user_id", Type: "INT"},
								},
							},
						},
					},
				},
			},
		},
	}

	queries, duplicates, err := sqlgenerator.GenerateQueryCreateTempTablePostgres(view, logger, "postgres")
	t.Log(duplicates)
	if err != nil {
		t.Fatalf("ошибка генерации: %v", err)
	}

	for _, duplicate := range duplicates {
		t.Logf("Duplicate %s:", duplicate)
	}

	for i, query := range queries.Queries {
		t.Logf("Query %d:\n%s", i+1, query)
	}
}

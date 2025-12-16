package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGeneratetInsertDataQuery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	view := models.View{
		Sources: []models.Source{
			{
				Name: "source_1",
				Schemas: []models.Schema{
					{
						Name: "schema_1",
						Tables: []models.Table{
							{
								Name: "temp_test_table",
								Columns: []models.Column{
									{Name: "id"},
									{Name: "name"},
									{Name: "active"},
									{Name: "created_at"},
								},
							},
						},
					},
				},
			},
		},
	}

	selectData := []map[string]interface{}{
		{
			"id":         1,
			"name":       "Alice",
			"active":     true,
			"created_at": time.Date(2023, 3, 15, 14, 30, 0, 0, time.UTC),
		},
		{
			"id":         2,
			"name":       "Bob",
			"active":     false,
			"created_at": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	query, err := sqlgenerator.GenerateInsertDataQuery(view, selectData, "temp_test_table", logger, "postgres")

	require.NoError(t, err)
	t.Log("Generated query:\n" + query.Query)

	require.Contains(t, query.Query, "INSERT INTO temp_test_table")
	require.Contains(t, query.Query, "TRUE")
	require.Contains(t, query.Query, "FALSE")
	require.Contains(t, query.Query, "'2023-03-15")
}

func TestGenerateInsertDataQuery_WithAlias(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	view := models.View{
		Sources: []models.Source{
			{
				Name: "source_1",
				Schemas: []models.Schema{
					{
						Name: "schema_1",
						Tables: []models.Table{
							{
								Name: "temp_test_table",
								Columns: []models.Column{
									{Name: "id", Alias: "route_id"},
									{Name: "title"},
								},
							},
						},
					},
				},
			},
		},
	}

	selectData := []map[string]interface{}{
		{
			"route_id": 123,
			"title":    "Route 123",
		},
	}

	query, err := sqlgenerator.GenerateInsertDataQuery(view, selectData, "temp_test_table", logger, "postgres")

	require.NoError(t, err)
	require.Contains(t, query.Query, "INSERT INTO temp_test_table (route_id, title) VALUES (123, 'Route 123')")
}

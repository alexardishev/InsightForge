package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestGenerateSelectInsertDataQuery_Success(t *testing.T) {
	view := models.View{
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
									{Name: "id"},
									{Name: "name"},
								},
							},
						},
					},
				},
			},
		},
	}

	query, err := sqlgenerator.GenerateSelectInsertDataQuery(view, 0, 100, "users", getTestLogger())
	t.Logf(query.Query)
	assert.NoError(t, err)
	assert.Equal(t, "users", query.TableName)
	assert.Equal(t, "db1", query.SourceName)
	assert.Contains(t, query.Query, "SELECT id, name FROM users OFFSET 0 LIMIT 100")
}

func TestGenerateSelectInsertDataQuery_DuplicateColumns(t *testing.T) {
	view := models.View{
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
									{Name: "id"},
									{Name: "id"}, // дубликат
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := sqlgenerator.GenerateSelectInsertDataQuery(view, 0, 50, "users", getTestLogger())

	assert.Error(t, err)
	assert.EqualError(t, err, "колонки с одинаковыми именами недопустимы")
}

func TestGenerateSelectInsertDataQuery_TableNotFound(t *testing.T) {
	view := models.View{
		Sources: []models.Source{
			{
				Name: "db1",
				Schemas: []models.Schema{
					{
						Name: "public",
						Tables: []models.Table{
							{
								Name: "customers",
								Columns: []models.Column{
									{Name: "id"},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := sqlgenerator.GenerateSelectInsertDataQuery(view, 0, 20, "orders", getTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "таблица orders не найдена")
}

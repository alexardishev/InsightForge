package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestGenerateSelectInsertDataQuery_WithJSONAndFieldTransform(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	view := models.View{
		Name: "test_view",
		Sources: []models.Source{
			{
				Name: "postgres",
				Schemas: []models.Schema{
					{
						Name: "public",
						Tables: []models.Table{
							{
								Name: "users",
								Columns: []models.Column{
									{
										Name:         "id",
										Type:         "uuid",
										IsPrimaryKey: true,
									},
									{
										Name: "email",
										Type: "text",
									},
									{
										Name: "metadata",
										Type: "jsonb",
										Transform: &models.Transform{
											Type: "JSON",
											Mapping: &models.Mapping{
												MappingJSON: []models.MappingJSON{
													{
														Mapping: map[string]string{
															"city": "user_city",
														},
														TypeField: "text",
													},
												},
											},
										},
									},
									{
										Name: "status",
										Type: "int",
										Transform: &models.Transform{
											Type: "FieldTransform",
											Mapping: &models.Mapping{
												Mapping: map[string]string{
													"1": "Создано",
													"2": "В обработке",
												},
												AliasNewColumnTransform: "status_label",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	query, err := sqlgenerator.GenerateSelectInsertDataQuery(view, 0, 100, "users", logger)
	t.Logf(query.Query)

	require.NoError(t, err)
	require.Contains(t, query.Query, "metadata->>'city' AS user_city")
	require.Contains(t, query.Query, "CASE WHEN status = '1' THEN 'Создано' WHEN status = '2' THEN 'В обработке' END as status_label")
	require.Contains(t, query.Query, "ORDER BY id")
	t.Logf(query.Query)

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

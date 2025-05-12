package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestCreateViewQuery_WithJoins(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	schema := models.View{
		Name: "user_basic_info",
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
									{Name: "id", Type: "uuid", IsPrimaryKey: true, IsNullable: true},
									{Name: "email", Type: "text", IsNullable: true},
									{
										Name:       "json_transform",
										Type:       "jsonb",
										IsNullable: true,
										Transform: &models.Transform{
											Type:         "JSON",
											Mode:         "Mapping",
											OutputColumn: "json_transform",
											Mapping: &models.Mapping{
												TypeMap: "JSON",
												MappingJSON: []models.MappingJSON{
													{
														Mapping:   map[string]string{"field1_in_json": "field1_view_column"},
														TypeField: "int",
													},
													{
														Mapping:   map[string]string{"field2_in_json": "field2_view_column"},
														TypeField: "text",
													},
												},
											},
										},
									},
									{
										Name:       "status",
										Type:       "int",
										IsNullable: false,
										Transform: &models.Transform{
											Type:         "FieldTransform",
											OutputColumn: "status_label",
											Mapping: &models.Mapping{
												TypeMap:                 "FieldTransform",
												AliasNewColumnTransform: "status_label",
												Mapping: map[string]string{
													"1": "Создан",
													"2": "В обработке",
													"3": "Завершен",
												},
											},
										},
									},
								},
							},
							{
								Name: "profiles",
								Columns: []models.Column{
									{Name: "user_id", Type: "uuid", IsPrimaryKey: true, IsNullable: false},
									{Name: "age", Type: "int", IsNullable: true},
								},
							},
						},
					},
				},
			},
		},
		Joins: []*models.Join{
			{
				Inner: &models.JoinSide{
					Source:       "postgres",
					Schema:       "public",
					Table:        "profiles",
					ColumnFirst:  "id",
					ColumnSecond: "user_id",
				},
			},
		},
	}

	viewJoin := models.ViewJoinTable{
		TempTables: []models.TempTable{
			{
				TempTableName: "temp_postgres_public_users",
				TempColumns: []models.TempColumn{
					{ColumnName: "id"},
					{ColumnName: "email"},
					{ColumnName: "json_transform"},
					{ColumnName: "status"},
					{ColumnName: "field1_view_column"},
					{ColumnName: "field2_view_column"},
					{ColumnName: "status_label"},
				},
			},
			{
				TempTableName: "temp_postgres_public_profiles",
				TempColumns: []models.TempColumn{
					{ColumnName: "user_id"},
					{ColumnName: "age"},
				},
			},
		},
	}

	result, err := sqlgenerator.CreateViewQuery1(schema, viewJoin, logger)
	t.Logf(result.Query)
	if err != nil {
		t.Fatalf("error generating view query: %v", err)
	}

	if !strings.Contains(result.Query, "JOIN temp_postgres_public_profiles") {
		t.Errorf("expected JOIN clause in query, got: %s", result.Query)
	}

	if !strings.Contains(result.Query, "CREATE TABLE user_basic_info") {
		t.Errorf("expected CREATE TABLE clause in query, got: %s", result.Query)
	}
}

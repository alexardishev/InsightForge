package sqlgenerator_test

import (
	"io"
	"log/slog"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"

	"github.com/stretchr/testify/require"
)

// transformView — таблица с JSON- и FieldTransform-трансформациями по четыре ключа:
// при обходе map в случайном порядке SQL почти наверняка будет разным от запуска к запуску.
func transformView() models.View {
	return models.View{Name: "v", Sources: []models.Source{{Name: "src", Schemas: []models.Schema{{
		Name: "public",
		Tables: []models.Table{{Name: "users", Columns: []models.Column{
			{Name: "id", Type: "int", IsPrimaryKey: true},
			{Name: "metadata", Type: "jsonb", Transform: &models.Transform{Type: "JSON", Mapping: &models.Mapping{
				TypeMap: "JSON",
				MappingJSON: []models.MappingJSON{{TypeField: "text", Mapping: map[string]string{
					"d": "col_d", "a": "col_a", "c": "col_c", "b": "col_b",
				}}},
			}}},
			{Name: "status", Type: "int", Transform: &models.Transform{Type: "FieldTransform", Mapping: &models.Mapping{
				TypeMap:                 "FieldTransform",
				AliasNewColumnTransform: "status_label",
				Mapping:                 map[string]string{"4": "w", "1": "x", "3": "z", "2": "y"},
			}}},
		}}},
	}}}}}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestSelectInsertDataQueryIsDeterministic(t *testing.T) {
	for _, db := range []string{sqlgenerator.DbPostgres, sqlgenerator.DbClickhouse} {
		t.Run(db, func(t *testing.T) {
			first, err := sqlgenerator.GenerateSelectInsertDataQuery(transformView(), 0, 100, "users", discardLogger(), db)
			require.NoError(t, err)
			for i := 0; i < 30; i++ {
				q, err := sqlgenerator.GenerateSelectInsertDataQuery(transformView(), 0, 100, "users", discardLogger(), db)
				require.NoError(t, err)
				require.Equal(t, first.Query, q.Query, "the same view must always give the same SQL")
			}
		})
	}
}

func TestSelectInsertDataQueryOrdersTransformsByKey(t *testing.T) {
	q, err := sqlgenerator.GenerateSelectInsertDataQuery(transformView(), 0, 100, "users", discardLogger(), sqlgenerator.DbPostgres)
	require.NoError(t, err)

	require.Contains(t, q.Query, "metadata->>'a' AS col_a, metadata->>'b' AS col_b, metadata->>'c' AS col_c, metadata->>'d' AS col_d")
	require.Contains(t, q.Query, "CASE WHEN status = '1' THEN 'x' WHEN status = '2' THEN 'y' WHEN status = '3' THEN 'z' WHEN status = '4' THEN 'w' END as status_label")
}

func TestCreateTempTableQueryIsDeterministic(t *testing.T) {
	for _, db := range []string{sqlgenerator.DbPostgres, sqlgenerator.DbClickhouse} {
		t.Run(db, func(t *testing.T) {
			view := transformView()
			first, _, err := sqlgenerator.GenerateQueryCreateTempTable(&view, discardLogger(), db)
			require.NoError(t, err)
			for i := 0; i < 30; i++ {
				view := transformView()
				q, _, err := sqlgenerator.GenerateQueryCreateTempTable(&view, discardLogger(), db)
				require.NoError(t, err)
				require.Equal(t, first, q, "column order of the temp table must not depend on map iteration")
			}
		})
	}
}

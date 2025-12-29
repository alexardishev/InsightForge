package sqlgenerator_test

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestCreateViewQuery_WithChainJoins(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	schema := models.View{
		Name: "user_basic_info",
		Joins: []*models.Join{
			{
				Inner: &models.JoinCondition{
					Left:  models.JoinEndpoint{Source: "postgres_oltp", Schema: "public", Table: "users", Column: "id"},
					Right: models.JoinEndpoint{Source: "postgres_copy", Schema: "public", Table: "profiles", Column: "user_id"},
				},
			},
			{
				Inner: &models.JoinCondition{
					Left:  models.JoinEndpoint{Source: "postgres_copy", Schema: "public", Table: "profiles", Column: "profile_id"},
					Right: models.JoinEndpoint{Source: "postgres_copy", Schema: "public", Table: "sessions", Column: "profile_id"},
				},
			},
		},
	}

	viewJoin := models.ViewJoinTable{
		TempTables: []models.TempTable{
			{
				TempTableName: "temp_postgres_oltp_public_users",
				Source:        "postgres_oltp",
				Schema:        "public",
				Table:         "users",
				TempColumns:   []models.TempColumn{{ColumnName: "id"}, {ColumnName: "email"}},
			},
			{
				TempTableName: "temp_postgres_copy_public_profiles",
				Source:        "postgres_copy",
				Schema:        "public",
				Table:         "profiles",
				TempColumns:   []models.TempColumn{{ColumnName: "profile_id"}, {ColumnName: "user_id"}},
			},
			{
				TempTableName: "temp_postgres_copy_public_sessions",
				Source:        "postgres_copy",
				Schema:        "public",
				Table:         "sessions",
				TempColumns:   []models.TempColumn{{ColumnName: "session_id"}, {ColumnName: "profile_id"}},
			},
		},
	}

	result, err := sqlgenerator.CreateViewQuery(schema, viewJoin, logger, "postgres")
	if err != nil {
		t.Fatalf("error generating view query: %v", err)
	}

	if !strings.Contains(result.Query, "JOIN \"temp_postgres_copy_public_profiles\" \"t2\" ON \"t1\".\"id\" = \"t2\".\"user_id\"") {
		t.Fatalf("expected chain join to profiles, got: %s", result.Query)
	}

	if !strings.Contains(result.Query, "JOIN \"temp_postgres_copy_public_sessions\" \"t3\" ON \"t2\".\"profile_id\" = \"t3\".\"profile_id\"") {
		t.Fatalf("expected chain join to sessions, got: %s", result.Query)
	}

	if !strings.Contains(result.Query, "FROM \"temp_postgres_oltp_public_users\" \"t1\"") {
		t.Fatalf("expected root from users, got: %s", result.Query)
	}
}

func TestCreateViewQuery_MissingTempTable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	schema := models.View{
		Name: "broken_view",
		Joins: []*models.Join{
			{Inner: &models.JoinCondition{Left: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "a", Column: "id"}, Right: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "b", Column: "a_id"}}},
		},
	}
	viewJoin := models.ViewJoinTable{TempTables: []models.TempTable{{TempTableName: "temp_db1_public_a", Source: "db1", Schema: "public", Table: "a"}}}

	_, err := sqlgenerator.CreateViewQuery(schema, viewJoin, logger, "postgres")
	if err == nil {
		t.Fatalf("expected error for missing temp table, got nil")
	}
}

func TestCreateViewQuery_DisconnectedGraph(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	schema := models.View{
		Name: "disconnected",
		Joins: []*models.Join{
			{Inner: &models.JoinCondition{Left: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "a", Column: "id"}, Right: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "b", Column: "a_id"}}},
			{Inner: &models.JoinCondition{Left: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "c", Column: "id"}, Right: models.JoinEndpoint{Source: "db1", Schema: "public", Table: "d", Column: "c_id"}}},
		},
	}
	viewJoin := models.ViewJoinTable{
		TempTables: []models.TempTable{
			{TempTableName: "temp_db1_public_a", Source: "db1", Schema: "public", Table: "a"},
			{TempTableName: "temp_db1_public_b", Source: "db1", Schema: "public", Table: "b"},
			{TempTableName: "temp_db1_public_c", Source: "db1", Schema: "public", Table: "c"},
			{TempTableName: "temp_db1_public_d", Source: "db1", Schema: "public", Table: "d"},
		},
	}

	_, err := sqlgenerator.CreateViewQuery(schema, viewJoin, logger, "postgres")
	if err == nil {
		t.Fatalf("expected error for disconnected joins")
	}
}

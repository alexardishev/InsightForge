package models

import "time"

const (
	ColumnMismatchStatusOpen     = "open"
	ColumnMismatchStatusResolved = "resolved"
)

const (
	ColumnMismatchTypeSchemaOnly   = "schema_only"
	ColumnMismatchTypeMissingInDWH = "missing_in_dwh"
	ColumnMismatchTypeDWHOnly      = "dwh_only"
	ColumnMismatchTypeRename       = "rename_candidate"
)

type ColumnMismatchGroup struct {
	ID           int64      `db:"id" json:"id"`
	SchemaID     int64      `db:"schema_id" json:"schema_id"`
	DatabaseName string     `db:"database_name" json:"database_name"`
	SchemaName   string     `db:"schema_name" json:"schema_name"`
	TableName    string     `db:"table_name" json:"table_name"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt   *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
}

type ColumnMismatchItem struct {
	ID            int64    `db:"id" json:"id"`
	GroupID       int64    `db:"group_id" json:"group_id"`
	OldColumnName *string  `db:"old_column_name" json:"old_column_name,omitempty"`
	NewColumnName *string  `db:"new_column_name" json:"new_column_name,omitempty"`
	Score         *float64 `db:"score" json:"score,omitempty"`
	Type          string   `db:"mismatch_type" json:"type"`
}

type ColumnMismatchGroupWithItems struct {
	Group ColumnMismatchGroup  `json:"group"`
	Items []ColumnMismatchItem `json:"items"`
}

type ColumnMismatchFilter struct {
	SchemaID     *int64
	DatabaseName *string
	SchemaName   *string
	TableName    *string
	Status       *string

	Limit  int
	Offset int
}

type ColumnMismatchResolution struct {
	Renames []RenameDecision `json:"renames"`
	Deletes []string         `json:"deletes"`
	Ignores []string         `json:"ignores"`
}

type RenameDecision struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

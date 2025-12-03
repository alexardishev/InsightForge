package models

import "time"

// ColumnMismatchGroup describes a set of column inconsistencies for a specific table/view.
type ColumnMismatchGroup struct {
	ID           int64      `db:"id" json:"id"`
	SchemaID     int64      `db:"schema_id" json:"schema_id"`
	DatabaseName string     `db:"database_name" json:"database_name"`
	SchemaName   string     `db:"schema_name" json:"schema_name"`
	TableName    string     `db:"table_name" json:"table_name"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt   *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`

	Items   []ColumnMismatchItem `json:"items,omitempty"`
	Missing []string             `json:"missing,omitempty"`
	Added   []string             `json:"added,omitempty"`
}

// ColumnMismatchItem represents either a single unmatched column or a candidate pair.
type ColumnMismatchItem struct {
	ID             int64    `db:"id" json:"id"`
	GroupID        int64    `db:"group_id" json:"group_id"`
	OldColumnName  *string  `db:"old_column_name" json:"old_column_name,omitempty"`
	NewColumnName  *string  `db:"new_column_name" json:"new_column_name,omitempty"`
	Score          *float64 `db:"score" json:"score,omitempty"`
	SuggestedMatch bool     `db:"suggested" json:"suggested"`
}

// ColumnMismatchGroupFilter defines query params for listing mismatch groups.
type ColumnMismatchGroupFilter struct {
	SchemaID     *int64
	DatabaseName *string
	SchemaName   *string
	TableName    *string
	Status       *string

	Limit               int
	Offset              int
	SortByCreatedAtDesc bool
}

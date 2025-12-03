package models

import "time"

type ColumnRenameSuggestion struct {
	ID            int64     `db:"id" json:"id"`
	SchemaID      int64     `db:"schema_id" json:"schema_id"`
	DatabaseName  string    `db:"database_name" json:"database_name"`
	SchemaName    string    `db:"schema_name" json:"schema_name"`
	TableName     string    `db:"table_name" json:"table_name"`
	OldColumnName string    `db:"old_column_name" json:"old_column_name"`
	NewColumnName string    `db:"new_column_name" json:"new_column_name"`
	Strategy      string    `db:"strategy" json:"strategy"`
	TaskNumber    *string   `db:"task_number" json:"task_number,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type ColumnRenameSuggestionFilter struct {
	SchemaID     *int64
	DatabaseName *string
	SchemaName   *string
	TableName    *string

	Limit               int
	Offset              int
	SortByCreatedAtDesc bool
}

package models

// SchemaInfo contains brief information about stored schemas.
type SchemaInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SourceCount int    `json:"source_count"`
	TableCount  int    `json:"table_count"`
}

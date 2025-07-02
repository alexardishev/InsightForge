package models

type Indexes struct {
	Indexes []Index
}
type Index struct {
	IndexName string `json:"index_name,omitempty"`
	IndexDef  string `json:"index_def,omitempty"`
}

type IndexTransferTable struct {
	TableName  string   `json:"table_name,omitempty"`
	SourceName string   `json:"source_name,omitempty"`
	SchemaName string   `json:"schema_name,omitempty"`
	Columns    []string `json:"columns,omitempty"`
}

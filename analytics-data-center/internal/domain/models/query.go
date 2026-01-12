package models

type Queries struct {
	Queries []Query
}

type Query struct {
	SourceName    string
	SchemaName    string
	TableName     string
	BaseTableName string
	Query         string
}

package models

type Queries struct {
	Queries []Query
}

type Query struct {
	SourceName string
	TableName  string
	Query      string
}

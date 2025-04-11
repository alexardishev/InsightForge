package models

type Queries struct {
	Queries []Query
}

type Query struct {
	TableName string
	Query     string
}

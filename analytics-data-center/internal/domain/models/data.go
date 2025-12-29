package models

type CountInsertData struct {
	Count         int64  `json:"count,omitempty"`
	TableName     string `json:"table_name,omitempty"`
	DataBaseName  string `json:"data_base_name,omitempty"`
	SchemaName    string `json:"schema_name,omitempty"`
	TempTableName string `json:"temp_table_name,omitempty"`
}

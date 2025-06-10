package models

type ViewJoinTable struct {
	TempTables []TempTable
}

type TempTable struct {
	TempTableName string
	TempColumns   []TempColumn
}

type TempColumn struct {
	ColumnName string
}

type ColumnInfo struct {
	ColumnName  string  `json:"column_name,omitempty"`
	Type        string  `json:"type,omitempty"`
	IsNullable  bool    `json:"is_nullable,omitempty"`
	Default     *string `json:"default,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPK        bool    `json:"is_pk,omitempty"`
	IsFK        bool    `json:"is_fk,omitempty"`
	IsUnique    bool    `json:"is_unique,omitempty"`
}

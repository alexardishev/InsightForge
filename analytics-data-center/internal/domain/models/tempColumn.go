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

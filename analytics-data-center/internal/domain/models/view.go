package models

type View struct {
	Name    string   `json:"view_name"`
	Sources []Source `json:"sources"`
	Joins   []*Join  `json:"joins"`
}
type Source struct {
	Name    string   `json:"name"`
	Schemas []Schema `json:"schemas"`
}

type Schema struct {
	Name   string  `json:"name"`
	Tables []Table `json:"tables"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type Column struct {
	Name         string     `json:"name,omitempty"`
	Alias        string     `json:"alias,omitempty"`
	IsUpdateKey  bool       `json:"is_update_key,omitempty"`
	Transform    *Transform `json:"transform,omitempty"`
	Reference    *Reference `json:"reference,omitempty"`
	Type         string     `json:"type,omitempty"`
	DataType     string     `json:"data_type,omitempty"`
	UdtSchema    string     `json:"udt_schema,omitempty"`
	UdtName      string     `json:"udt_name,omitempty"`
	CharMaxLen   *int64     `json:"char_max_len,omitempty"`
	NumPrecision *int64     `json:"num_precision,omitempty"`
	NumScale     *int64     `json:"num_scale,omitempty"`
	IsDeleted    bool       `json:"is_deleted,omitempty"`
	IsNullable   bool       `json:"is_nullable,omitempty"`
	IsPrimaryKey bool       `json:"is_primary_key,omitempty"`
	ViewKey      string     `json:"view_key,omitempty"`
	IsFK         bool       `json:"is_fk,omitempty"`
	Default      string     `json:"default,omitempty"`
	IsUNQ        bool       `json:"is_unq,omitempty"`
}

type Transform struct {
	Type         string   `json:"type"`
	Mode         string   `json:"mode"`
	OutputColumn string   `json:"output_column"`
	Mapping      *Mapping `json:"mapping"`
}

type Mapping struct {
	TypeMap                 string            `json:"type_map,omitempty"` // либо JSON либо FieldTransform
	Mapping                 map[string]string `json:"mapping,omitempty"`  // ключ это значение в БД, значение это трансформированное значение
	AliasNewColumnTransform string            `json:"alias_new_column_transform,omitempty"`
	TypeField               string            `json:"type_field,omitempty"` // тип поля трансформации
	MappingJSON             []MappingJSON     `json:"mapping_json,omitempty"`
}

type MappingJSON struct {
	Mapping   map[string]string `json:"mapping,omitempty"` // если тип JSON тогда ключ это Имя поля в JSON, value это наименование колонки во вью и темп таблице
	TypeField string            `json:"type_field,omitempty"`
}

type Reference struct {
	Source string `json:"source"`
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Join struct {
	Inner *JoinSide `json:"inner"`
	// TO DO сделать другие джоины потом
	// Left  JoinSide `json:"left"`
	// Right JoinSide `json:"right"`
}

type JoinSide struct {
	Source       string `json:"source,omitempty"`
	Schema       string `json:"schema,omitempty"`
	Table        string `json:"table,omitempty"`
	MainTable    string `json:"main_table,omitempty"`
	ColumnFirst  string `json:"column_first,omitempty"`
	ColumnSecond string `json:"column_second,omitempty"`
}

package models

type View struct {
	Name    string   `json:"view_name"`
	Sources []Source `json:"sources"`
	Joins   []Join   `json:"joins"`
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
	IsDeleted    bool       `json:"is_deleted,omitempty"`
	IsNullable   bool       `json:"is_nullable,omitempty"`
	IsPrimaryKey bool       `json:"is_primary_key,omitempty"`
}

type Transform struct {
	Type         string  `json:"type"`
	Mode         string  `json:"mode"`
	OutputColumn string  `json:"output_column"`
	Mapping      Mapping `json:"mapping"`
}

type Mapping struct {
	TypeMap string            // либо JSON либо FieldTransform
	Mapping map[string]string // если тип JSON тогда ключ это Имя поля в JSON, value это наименование колонки во вью и темп таблице
}

type Reference struct {
	Source string `json:"source"`
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

type Join struct {
	Left  JoinSide `json:"left"`
	Right JoinSide `json:"right"`
}

type JoinSide struct {
	Source string `json:"source"`
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

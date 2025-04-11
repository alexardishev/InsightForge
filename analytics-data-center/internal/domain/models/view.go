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
	Name        string     `json:"name"`
	Alias       string     `json:"alias,omitempty"`
	IsUpdateKey bool       `json:"is_update_key,omitempty"`
	Transform   *Transform `json:"transform,omitempty"`
	Reference   *Reference `json:"reference,omitempty"`
	Type        string     `json:"type"`
}

type Transform struct {
	Type         string            `json:"type"`
	Mode         string            `json:"mode"`
	OutputColumn string            `json:"output_column"`
	Mapping      map[string]string `json:"mapping"`
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

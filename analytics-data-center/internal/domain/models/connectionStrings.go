package models

type ConnectionStrings struct {
	ConnectionStrings []ConnectionString `validate:"required" json:"connection_strings"`
}

type ConnectionString struct {
	ConnectionString map[string]string `validate:"required" json:"connection_string"`
}

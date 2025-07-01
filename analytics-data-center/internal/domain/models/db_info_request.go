package models

// DBInfoRequest represents request for getting database info with pagination.
type DBInfoRequest struct {
	ConnectionStrings []ConnectionString `json:"connection_strings" validate:"required"`
	Page              int                `json:"page"`
	PageSize          int                `json:"page_size"`
}

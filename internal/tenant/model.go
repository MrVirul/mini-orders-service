package tenant

import "database/sql"

type RequestTenant struct {
	Code     string
	SiteName string
	Currency string
	DB       *sql.DB
}

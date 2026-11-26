package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when no tenant has the given code.
	ErrNotFound = errors.New("tenant not found")
	// ErrInactive is returned when the tenant exists but is suspended.
	ErrInactive = errors.New("tenant is inactive")
)

// Tenant is one row of the master registry: where a tenant's database lives.
type Tenant struct {
	ID           uint64
	Code         string
	SiteName     string
	Host         string
	Port         string
	DatabaseName string
	Username     string
	// Password is the tenant database credential. It is never serialized.
	Password string `json:"-"`
	Currency string
	IsActive bool
}

// String omits the password so a Tenant is safe to log with %v or %+v.
func (t Tenant) String() string {
	return fmt.Sprintf("Tenant{Code: %q, Database: %q}", t.Code, t.DatabaseName)
}

const findByCodeQuery = `
SELECT id, x_tenant_code, site_name, host, port, database_name,
       username, password, currency, is_active
FROM tenants
WHERE x_tenant_code = ?`

// FindByCode looks up a tenant by its X-Tenant-Code value.
//
// It returns ErrNotFound when no row matches, ErrInactive when the tenant is
// suspended, and a wrapped error for any other database failure.
func FindByCode(ctx context.Context, db *sql.DB, code string) (*Tenant, error) {
	var t Tenant

	err := db.QueryRowContext(ctx, findByCodeQuery, code).Scan(
		&t.ID,
		&t.Code,
		&t.SiteName,
		&t.Host,
		&t.Port,
		&t.DatabaseName,
		&t.Username,
		&t.Password,
		&t.Currency,
		&t.IsActive,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, fmt.Errorf("find tenant by code: %w", err)
	case !t.IsActive:
		return nil, ErrInactive
	}

	return &t, nil
}

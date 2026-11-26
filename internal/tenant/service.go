package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrTenantDBUnavailable = errors.New("tenant database unavailable")

type Finder interface {
	FindByCode(ctx context.Context, code string) (*Tenant, error)
}
type PoolProvider interface {
	Get(ctx context.Context, t *Tenant) (*sql.DB, error)
}

type Service struct {
	repo  Finder
	pools PoolProvider
}

func NewService(repo Finder, pools PoolProvider) *Service {
	return &Service{repo: repo, pools: pools}
}

func (s *Service) Resolve(ctx context.Context, code string) (*RequestTenant, error) {
	t, err := s.repo.FindByCode(ctx, code) // ErrNotFound / ErrInactive come straight through
	if err != nil {
		return nil, err
	}
	db, err := s.pools.Get(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantDBUnavailable, err)
	}
	return &RequestTenant{Code: t.Code, SiteName: t.SiteName, Currency: t.Currency, DB: db}, nil
}

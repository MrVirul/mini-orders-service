package service

import (
	"context"

	"github.com/MrVirul/mini-orders-service/internal/database"
)

type MenuItemLister interface {
	ListAvailable(ctx context.Context) ([]database.MenuItem, error)
}

type MenuService struct {
	repo MenuItemLister
}

func NewMenuService(repo MenuItemLister) *MenuService {
	return &MenuService{repo: repo}
}

// List returns the items customers can currently order.
func (s *MenuService) List(ctx context.Context) ([]database.MenuItem, error) {
	return s.repo.ListAvailable(ctx)
}

package database

import (
	"context"
	"database/sql"
	"fmt"
)

type MenuItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	PriceMinor  int64   `json:"price_minor"`
}

type MenuRepository struct {
	db *sql.DB
}

func NewMenuRepository(db *sql.DB) *MenuRepository {
	return &MenuRepository{db: db}
}

func (r *MenuRepository) ListAvailable(ctx context.Context) ([]MenuItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, price_minor
		   FROM menu_items
		  WHERE is_available = 1
		  ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query menu items: %w", err)
	}
	defer rows.Close()

	items := []MenuItem{}
	for rows.Next() {
		var it MenuItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.PriceMinor); err != nil {
			return nil, fmt.Errorf("scan menu item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate menu items: %w", err)
	}
	return items, nil
}

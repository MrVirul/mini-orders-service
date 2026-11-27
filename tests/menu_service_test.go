package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/MrVirul/mini-orders-service/internal/app"
	"github.com/MrVirul/mini-orders-service/internal/database"
	"github.com/MrVirul/mini-orders-service/internal/service"
	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

func TestRepository_ListAvailable_FiltersUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery(`is_available = 1`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "price_minor"}).
			AddRow(1, "Margherita", "Tomato", 899).
			AddRow(3, "Garlic Bread", nil, 399))

	items, err := service.NewMenuService(database.NewMenuRepository(db)).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Description != nil || items[0].PriceMinor != 899 {
		t.Fatalf("unexpected items: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMenuItems_TwoTenants(t *testing.T) {
	pizzaDB, pizzaMock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer pizzaDB.Close()

	sushiDB, sushiMock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sushiDB.Close()

	pizzaMock.ExpectQuery(`is_available = 1`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "price_minor"}).
			AddRow(1, "Margherita", "Tomato, mozzarella, basil", 899).
			AddRow(3, "Garlic Bread", nil, 399),
	)
	sushiMock.ExpectQuery(`is_available = 1`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "price_minor"}).
			AddRow(1, "Salmon Nigiri", "2 pieces", 650).
			AddRow(3, "Miso Soup", nil, 350),
	)

	resolver := fakeResolver{tenants: map[string]*tenant.RequestTenant{
		"pizza_palace": {Code: "pizza_palace", SiteName: "Pizza Palace", Currency: "GBP", DB: pizzaDB},
		"sushi_house":  {Code: "sushi_house", SiteName: "Sushi House", Currency: "EUR", DB: sushiDB},
	}}
	srv := app.New(fakePinger{}, resolver)

	cases := []struct {
		tenantCode string
		wantFirst  string
	}{
		{"pizza_palace", "Margherita"},
		{"sushi_house", "Salmon Nigiri"},
	}

	for _, tc := range cases {
		t.Run(tc.tenantCode, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/menu-items", nil)
			req.Header.Set("X-Tenant-Code", tc.tenantCode)

			resp, err := srv.Test(req, -1)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var body struct {
				Items []database.MenuItem `json:"items"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(body.Items) != 2 || body.Items[0].Name != tc.wantFirst {
				t.Fatalf("items = %+v, want first item %q", body.Items, tc.wantFirst)
			}
		})
	}

	if err := pizzaMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("pizza mock: %v", err)
	}
	if err := sushiMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sushi mock: %v", err)
	}
}

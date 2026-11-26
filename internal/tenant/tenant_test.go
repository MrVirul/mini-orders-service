package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var columns = []string{
	"id", "x_tenant_code", "site_name", "host", "port",
	"database_name", "username", "password", "currency", "is_active",
}

func TestFindByCode_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM tenants").
		WithArgs("pizza_palace").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			1, "pizza_palace", "Pizza Palace", "127.0.0.1", "3307",
			"tenant_pizza_palace", "pizza_palace_user", "pw", "GBP", true,
		))

	got, err := FindByCode(context.Background(), db, "pizza_palace")
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	if got.Code != "pizza_palace" || got.DatabaseName != "tenant_pizza_palace" ||
		got.Port != "3307" || got.Currency != "GBP" || !got.IsActive {
		t.Fatalf("unexpected tenant: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindByCode_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM tenants").
		WithArgs("nope").
		WillReturnRows(sqlmock.NewRows(columns))

	got, err := FindByCode(context.Background(), db, "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if got != nil {
		t.Fatalf("tenant = %+v, want nil", got)
	}
}

func TestFindByCode_Inactive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM tenants").
		WithArgs("sushi_house").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			2, "sushi_house", "Sushi House", "127.0.0.1", "3308",
			"tenant_sushi_house", "sushi_house_user", "pw", "EUR", false,
		))

	got, err := FindByCode(context.Background(), db, "sushi_house")
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("err = %v, want ErrInactive", err)
	}
	if got != nil {
		t.Fatalf("tenant = %+v, want nil", got)
	}
}

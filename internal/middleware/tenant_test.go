package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

type fakeResolver struct {
	tenants map[string]*tenant.RequestTenant
	err     error
}

func (f fakeResolver) Resolve(_ context.Context, code string) (*tenant.RequestTenant, error) {
	if f.err != nil {
		return nil, f.err
	}
	if t, ok := f.tenants[code]; ok {
		return t, nil
	}
	return nil, tenant.ErrNotFound
}

func do(t *testing.T, r Resolver, header *string) (int, map[string]any) {
	t.Helper()
	app := fiber.New()
	app.Use(Tenant(r))
	app.Get("/tenant", func(c *fiber.Ctx) error {
		tn, _ := TenantFrom(c)
		return c.JSON(fiber.Map{"code": tn.Code, "currency": tn.Currency})
	})

	req := httptest.NewRequest(http.MethodGet, "/tenant", nil)
	if header != nil {
		req.Header.Set("X-Tenant-Code", *header)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

func str(s string) *string { return &s }

func errCode(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func TestTenantMiddleware(t *testing.T) {
	good := fakeResolver{tenants: map[string]*tenant.RequestTenant{
		"pizza_palace": {Code: "pizza_palace", Currency: "GBP"},
		"sushi_house":  {Code: "sushi_house", Currency: "EUR"},
	}}

	tests := []struct {
		name     string
		resolver Resolver
		header   *string
		status   int
		errCode  string
		currency string
	}{
		{"missing header", good, nil, 400, "tenant_header_missing", ""},
		{"blank header", good, str("   "), 400, "tenant_header_missing", ""},
		{"unknown tenant", good, str("nope"), 401, "tenant_unknown", ""},
		{"inactive tenant", fakeResolver{err: tenant.ErrInactive}, str("x"), 401, "tenant_unknown", ""},
		{"tenant db down", fakeResolver{err: tenant.ErrTenantDBUnavailable}, str("pizza_palace"), 503, "service_unavailable", ""},
		{"pizza", good, str("pizza_palace"), 200, "", "GBP"},
		{"sushi", good, str("sushi_house"), 200, "", "EUR"},
		{"header is trimmed", good, str(" sushi_house "), 200, "", "EUR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body := do(t, tc.resolver, tc.header)
			if status != tc.status {
				t.Fatalf("status = %d, want %d (%v)", status, tc.status, body)
			}
			if got := errCode(body); got != tc.errCode {
				t.Fatalf("error code = %q, want %q", got, tc.errCode)
			}
			if tc.currency != "" && body["currency"] != tc.currency {
				t.Fatalf("currency = %v, want %s", body["currency"], tc.currency)
			}
		})
	}
}

func TestTenantMiddleware_503DoesNotLeakDetails(t *testing.T) {
	r := fakeResolver{err: errors.New("dial tcp: user:s3cret@host refused")}
	_, body := do(t, r, str("pizza_palace"))
	b, _ := json.Marshal(body)
	if string(b) == "" || strings.Contains(string(b), "s3cret") {
		t.Fatalf("response leaked internal error: %s", b)
	}
}

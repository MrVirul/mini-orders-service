package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"

	"github.com/MrVirul/mini-orders-service/internal/httpx"
	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

const localsKey = "tenant"

type Resolver interface {
	Resolve(ctx context.Context, code string) (*tenant.RequestTenant, error)
}

func Tenant(r Resolver) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// copy the header: Fiber reuses this memory after the request ends
		code := strings.TrimSpace(utils.CopyString(c.Get("X-Tenant-Code")))
		if code == "" {
			return httpx.Error(c, fiber.StatusBadRequest,
				"tenant_header_missing", "Missing X-Tenant-Code header")
		}

		t, err := r.Resolve(c.Context(), code)
		switch {
		case err == nil:
			// fine, continue below
		case errors.Is(err, tenant.ErrNotFound), errors.Is(err, tenant.ErrInactive):
			return httpx.Error(c, fiber.StatusUnauthorized,
				"tenant_unknown", "Unknown tenant")
		default:
			slog.Error("tenant resolve failed", "tenant_code", code, "error", err)
			return httpx.Error(c, fiber.StatusServiceUnavailable,
				"service_unavailable", "Service temporarily unavailable")
		}

		c.Locals(localsKey, t)
		return c.Next()
	}
}

func TenantFrom(c *fiber.Ctx) (*tenant.RequestTenant, bool) {
	t, ok := c.Locals(localsKey).(*tenant.RequestTenant)
	return t, ok
}

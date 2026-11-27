package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger logs one line per request. It runs before Tenant so it also
// records rejected requests; tenant_code is read after the chain finishes and
// is empty when no tenant was resolved. No headers, bodies or credentials.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		tenantCode := ""
		if t, ok := TenantFrom(c); ok {
			tenantCode = t.Code
		}
		slog.Info("request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"tenant_code", tenantCode,
		)
		return err
	}
}

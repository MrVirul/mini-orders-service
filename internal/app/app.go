package app

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/MrVirul/mini-orders-service/internal/handler"
	"github.com/MrVirul/mini-orders-service/internal/httpx"
	"github.com/MrVirul/mini-orders-service/internal/middleware"
)

const healthPingTimeout = 2 * time.Second

// Pinger is satisfied by *sql.DB; it lets tests stub the master connection.
type Pinger interface {
	PingContext(ctx context.Context) error
}

func New(master Pinger, resolver middleware.Resolver) *fiber.App {
	app := fiber.New()
	app.Use(middleware.RequestLogger())

	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), healthPingTimeout)
		defer cancel()

		if err := master.PingContext(ctx); err != nil {
			log.Printf("health check: master database ping failed: %v", err)

			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unavailable",
				"master": "down",
			})
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"master": "up",
		})
	})

	// Everything registered below this line requires a valid X-Tenant-Code.
	// /health is registered above, so it works without the header.
	app.Use(middleware.Tenant(resolver))

	app.Get("/tenant", func(c *fiber.Ctx) error {
		t, ok := middleware.TenantFrom(c)
		if !ok {
			return httpx.Error(c, fiber.StatusInternalServerError,
				"internal_error", "Tenant missing from request context")
		}
		return c.JSON(fiber.Map{
			"code":      t.Code,
			"site_name": t.SiteName,
			"currency":  t.Currency,
		})
	})

	app.Get("/menu-items", handler.NewMenuHandler().List)

	return app
}

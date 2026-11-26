package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/MrVirul/mini-orders-service/internal/config"
	"github.com/MrVirul/mini-orders-service/internal/database"
	"github.com/MrVirul/mini-orders-service/internal/httpx"
	"github.com/MrVirul/mini-orders-service/internal/middleware"
	"github.com/MrVirul/mini-orders-service/internal/tenant"
)

const (
	port            = "3000"
	shutdownTimeout = 10 * time.Second

	healthPingTimeout = 2 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.Connect(cfg.MasterDSN)
	if err != nil {
		return fmt.Errorf("connect to master database: %w", err)
	}
	defer db.Close()

	log.Println("connected to master database")

	pools := database.NewPoolManager()
	defer pools.Close()

	app := newApp(db, tenant.NewService(tenant.NewRepository(db), pools))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		errCh <- app.Listen(":" + port)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}

		return nil

	case <-ctx.Done():
		log.Println("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		log.Println("server shutdown complete")
		return nil
	}
}

// pinger is satisfied by *sql.DB; it lets tests stub the master connection.
type pinger interface {
	PingContext(ctx context.Context) error
}

func newApp(master pinger, resolver middleware.Resolver) *fiber.App {
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

	return app
}

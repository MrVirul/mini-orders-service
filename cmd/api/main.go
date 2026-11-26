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

	app := newApp(db)

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

func newApp(master pinger) *fiber.App {
	app := fiber.New()

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

	return app
}

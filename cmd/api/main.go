package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	port            = "3000"
	shutdownTimeout = 10 * time.Second
)

func main() {
	app := newApp()

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
			log.Fatalf("http server: %v", err)
		}
	case <-ctx.Done():
		log.Println("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
}

func newApp() *fiber.App {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	return app
}

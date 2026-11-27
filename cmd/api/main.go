package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrVirul/mini-orders-service/internal/app"
	"github.com/MrVirul/mini-orders-service/internal/config"
	"github.com/MrVirul/mini-orders-service/internal/database"
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

	srv := app.New(db, tenant.NewService(tenant.NewRepository(db), pools))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Listen(":" + port)
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

		if err := srv.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		log.Println("server shutdown complete")
		return nil
	}
}

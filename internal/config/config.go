package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Config holds the settings the API needs to start.
type Config struct {
	// MasterDSN is the MySQL DSN for the read-only master registry connection.
	MasterDSN string
}

// Load reads .env (when present) and the process environment, then builds the
// master database DSN. Real environment variables win over .env values.
func Load() (*Config, error) {
	// .env is optional in production, where variables are set by the platform.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	user := os.Getenv("MASTER_DB_USER")
	password := os.Getenv("MASTER_DB_PASSWORD")
	if user == "" || password == "" {
		return nil, errors.New("MASTER_DB_USER and MASTER_DB_PASSWORD must be set")
	}

	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = user
	mysqlCfg.Passwd = password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = net.JoinHostPort(
		envOrDefault("MASTER_DB_HOST", "127.0.0.1"),
		envOrDefault("MASTER_DB_PORT", "3306"),
	)
	mysqlCfg.DBName = envOrDefault("MASTER_DB_NAME", "mini_master")
	mysqlCfg.ParseTime = true
	mysqlCfg.Loc = time.UTC
	mysqlCfg.Params = map[string]string{"charset": "utf8mb4"}

	return &Config{MasterDSN: mysqlCfg.FormatDSN()}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

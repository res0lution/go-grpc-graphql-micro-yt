package db

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"identity-manager/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPingTimeout = 5 * time.Second

func NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	if err := validateDatabaseConfig(cfg); err != nil {
		return nil, err
	}

	connString := buildPostgresConnString(cfg)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pg config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pg pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping pg: %w", err)
	}

	return pool, nil
}

func validateDatabaseConfig(cfg config.DatabaseConfig) error {
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("db host is required")
	}
	if strings.TrimSpace(cfg.Port) == "" {
		return fmt.Errorf("db port is required")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return fmt.Errorf("db user is required")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("db name is required")
	}

	return nil
}

func buildPostgresConnString(cfg config.DatabaseConfig) string {
	query := url.Values{}
	if sslMode := strings.TrimSpace(cfg.SSLMode); sslMode != "" {
		query.Set("sslmode", sslMode)
	}

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.Name,
		RawQuery: query.Encode(),
	}).String()
}

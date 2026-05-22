package database

import (
	"context"
	"fmt"

	"portal-core/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type Db struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
	config *config.Inventory
}

func New(cfg *config.Config) (*Db, error) {
	ctx := context.Background()

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Inventory.User,
		cfg.Inventory.Password,
		cfg.Inventory.Host,
		cfg.Inventory.Port,
		cfg.Inventory.DbName,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Db{
		pool:   pool,
		config: &cfg.Inventory,
	}, nil
}

func (db *Db) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *Db) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *Db) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *Db) Config() *config.Inventory {
	return db.config
}

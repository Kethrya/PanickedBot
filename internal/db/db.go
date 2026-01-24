package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Open(cfg Config) (*sqlx.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("db dsn is empty")
	}

	db, err := sqlx.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Connection pool tuning
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Enforce UTC session timezone (optional but strongly recommended)
	// You can remove this if you prefer server local time, but UTC reduces pain.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "SET time_zone = '+00:00'"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set time_zone: %w", err)
	}

	// Make sure foreign keys work (InnoDB will enforce; this is just a sanity check)
	// Also ensures the connection is valid now, not later.
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}


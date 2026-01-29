package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
	
	sqlcdb "PanickedBot/internal/db/sqlc"
)

type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DB wraps both the sqlx.DB and the sqlc Queries
type DB struct {
	*sqlx.DB
	Queries *sqlcdb.Queries
}

func Open(cfg Config) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("db dsn is empty")
	}

	sqlxDB, err := sqlx.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Connection pool tuning
	if cfg.MaxOpenConns > 0 {
		sqlxDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlxDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlxDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Enforce UTC session timezone (optional but strongly recommended)
	// You can remove this if you prefer server local time, but UTC reduces pain.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := sqlxDB.ExecContext(ctx, "SET time_zone = '+00:00'"); err != nil {
		_ = sqlxDB.Close()
		return nil, fmt.Errorf("set time_zone: %w", err)
	}

	// Make sure foreign keys work (InnoDB will enforce; this is just a sanity check)
	// Also ensures the connection is valid now, not later.
	if err := sqlxDB.PingContext(ctx); err != nil {
		_ = sqlxDB.Close()
		return nil, err
	}

	// Create sqlc Queries instance
	queries := sqlcdb.New(sqlxDB.DB)

	return &DB{
		DB:      sqlxDB,
		Queries: queries,
	}, nil
}

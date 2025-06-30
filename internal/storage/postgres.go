package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool DBPool

type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Close()
}

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return err
	}

	// Try a ping or simple query
	err = pool.Ping(context.Background())
	if err != nil {
		return err
	}

	Pool = pool
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

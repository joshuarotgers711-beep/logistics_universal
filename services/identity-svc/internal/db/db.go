package db

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct{ Pool *pgxpool.Pool }

func Connect(ctx context.Context) (*DB, error) {
	// Support secrets-file pattern first
	if f := os.Getenv("DATABASE_URL_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			if s := strings.TrimSpace(string(b)); s != "" {
				url := s
				pool, err := pgxpool.New(ctx, url)
				if err != nil {
					return nil, err
				}
				if err := pool.Ping(ctx); err != nil {
					return nil, err
				}
				log.Println("identity-svc connected to postgres")
				return &DB{Pool: pool}, nil
			}
		}
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@postgres:5432/lmp?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	log.Println("identity-svc connected to postgres")
	return &DB{Pool: pool}, nil
}

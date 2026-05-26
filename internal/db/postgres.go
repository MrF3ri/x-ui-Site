package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver
)

// NewPostgres opens a PostgreSQL connection via pgx stdlib and retries
// until the database is reachable or the attempt budget is exhausted.
func NewPostgres(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("empty DATABASE_DSN")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	const maxAttempts = 15
	for i := 1; i <= maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pingErr := db.PingContext(ctx)
		cancel()
		if pingErr == nil {
			log.Printf("db: postgres ready after %d attempt(s)", i)
			return db, nil
		}
		log.Printf("db: postgres not ready (attempt %d/%d): %v", i, maxAttempts, pingErr)
		time.Sleep(time.Duration(i) * 500 * time.Millisecond)
	}

	return nil, errors.New("postgres did not become ready in time")
}

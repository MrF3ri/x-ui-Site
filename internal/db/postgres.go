package db

import (
	"database/sql"
	"fmt"
	"time"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	var lastErr error
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				return db, nil
			}
			_ = db.Close()
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("postgres connection failed after retries: %w", lastErr)
}

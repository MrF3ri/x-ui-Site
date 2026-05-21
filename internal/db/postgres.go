package db

import (
	"database/sql"
	"errors"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("empty DATABASE_DSN")
	}
	return nil, errors.New("postgres driver not linked in this offline build; provide pgx stdlib in deployment build")
}

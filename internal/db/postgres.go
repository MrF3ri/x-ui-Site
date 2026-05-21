package db

import "database/sql"

func NewPostgres(dsn string) (*sql.DB, error) {
	// Driverless build-safe fallback: DB is optional at bootstrap in this phase.
	_ = dsn
	return nil, nil
}

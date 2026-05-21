package db

import "database/sql"

func NewPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil { return nil, err }
	return db, db.Ping()
}

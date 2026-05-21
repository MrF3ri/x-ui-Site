package db

import "errors"

type PostgresPool struct { DSN string; Connected bool }

func NewPostgres(dsn string) (*PostgresPool, error) {
	if dsn == "" { return nil, errors.New("empty dsn") }
	return &PostgresPool{DSN: dsn, Connected: true}, nil
}
func (p *PostgresPool) Close() error { p.Connected = false; return nil }

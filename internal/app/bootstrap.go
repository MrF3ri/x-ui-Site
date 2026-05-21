package app

import (
	"garudapanel/internal/config"
	"garudapanel/internal/db"
	httpserver "garudapanel/internal/http"
)

func Run() error {
	cfg := config.Load()
	pool, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil { return err }
	defer pool.Close()
	srv := httpserver.New(cfg.AppPort, cfg.JWTSecret)
	return srv.Start()
}

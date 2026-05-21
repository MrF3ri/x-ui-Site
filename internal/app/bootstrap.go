package app

import (
	"garudapanel/internal/config"
	"garudapanel/internal/db"
	httpserver "garudapanel/internal/http"
)

func Run() error {
	cfg := config.Load()
	dbConn, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil { return err }
	defer dbConn.Close()
	if err := db.RunMigrations(dbConn, "migrations"); err != nil { return err }
	srv := httpserver.New(cfg.AppPort, dbConn, cfg.JWTSecret)
	return srv.Start()
}

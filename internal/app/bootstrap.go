package app

import (
	"log"
	"os"

	"garudapanel/internal/config"
	"garudapanel/internal/db"
	"garudapanel/internal/storefront"
	httpserver "garudapanel/internal/http"
)

func Run() error {
	cfg := config.Load()
	dbConn, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil { log.Printf("database init warning: %v", err) }
	if dbConn != nil {
		defer dbConn.Close()
		if err := db.RunMigrations(dbConn, "migrations"); err != nil { log.Printf("migration warning: %v", err) }
		if os.Getenv("APP_ENV") == "development" { _ = storefront.New(dbConn).SeedDemo() }
	}
	srv := httpserver.New(cfg.AppPort, dbConn, cfg.JWTSecret)
	return srv.Start()
}

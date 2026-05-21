package app

import (
	"log"

	"garudapanel/internal/config"
	"garudapanel/internal/db"
	"garudapanel/internal/storefront"
	httpserver "garudapanel/internal/http"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Printf("boot env=%s", cfg.AppEnv)
	dbConn, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	log.Printf("postgres connected")
	defer dbConn.Close()
	if err := db.RunMigrations(dbConn, "migrations"); err != nil {
		return err
	}
	log.Printf("migrations executed")
	if cfg.AppEnv == "development" {
		_ = storefront.New(dbConn).SeedDemo()
	}
	srv := httpserver.New(cfg.AppPort, dbConn, cfg.JWTSecret, cfg.AppEnv, cfg.RedisAddr, cfg.MinIOEndpoint)
	log.Printf("services initialized")
	log.Printf("http listening on :%s", cfg.AppPort)
	return srv.Start()
}

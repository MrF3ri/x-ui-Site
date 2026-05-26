package app

import (
	"log"

	"garudapanel/internal/config"
	"garudapanel/internal/db"
	httpserver "garudapanel/internal/http"
	"garudapanel/internal/storefront"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("fatal config error: %v", err)
		return err
	}
	log.Printf("boot env=%s", cfg.AppEnv)
	dbConn, err := db.NewPostgres(cfg.DatabaseDSN)
	if err != nil {
		log.Printf("fatal db connect error: %v", err)
		return err
	}
	log.Printf("postgres connected")
	defer dbConn.Close()

	if err := db.RunMigrations(dbConn, "migrations"); err != nil {
		log.Printf("fatal migration error: %v", err)
		return err
	}
	log.Printf("migrations complete")

	if cfg.AppEnv == "development" {
		if err := storefront.New(dbConn).SeedDemo(); err != nil {
			log.Printf("fatal seed error: %v", err)
			return err
		}
		log.Printf("seed complete")
	}

	srv := httpserver.New(cfg.AppPort, dbConn, cfg.JWTSecret, cfg.PanelEncryptionKey, cfg.AppEnv, cfg.RedisAddr, cfg.MinIOEndpoint)
	log.Printf("services initialized")
	log.Printf("http listening on :%s", cfg.AppPort)
	return srv.Start()
}

package app

import (
	"errors"
	"log"
	stdhttp "net/http"

	"garudapanel/internal/config"
	httpserver "garudapanel/internal/http"
)

func Run() error {
	cfg := config.Load()
	srv := httpserver.New(cfg.AppPort)
	log.Printf("garudapanel booting on :%s mode=%s", cfg.AppPort, cfg.ServerMode)
	err := srv.Start()
	if errors.Is(err, stdhttp.ErrServerClosed) {
		return nil
	}
	return err
}

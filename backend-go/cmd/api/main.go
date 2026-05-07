package main

import (
	"log"

	"digidocs-mgt/backend-go/internal/app"
	"digidocs-mgt/backend-go/internal/config"
)

type httpServer interface {
	ListenAndServe() error
}

func main() {
	if err := run(config.Load, func(cfg config.Config) (httpServer, error) {
		return app.NewServer(cfg)
	}); err != nil {
		log.Fatal(err)
	}
}

func run(loadConfig func() config.Config, newServer func(config.Config) (httpServer, error)) error {
	cfg := loadConfig()

	server, err := newServer(cfg)
	if err != nil {
		return err
	}

	log.Printf("starting %s on %s", cfg.AppName, cfg.HTTPAddr)

	return server.ListenAndServe()
}

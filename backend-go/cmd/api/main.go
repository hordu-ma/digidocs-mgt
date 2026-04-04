package main

import (
	"log"

	"digidocs-mgt/backend-go/internal/app"
	"digidocs-mgt/backend-go/internal/config"
)

func main() {
	cfg := config.Load()

	server, err := app.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting %s on %s", cfg.AppName, cfg.HTTPAddr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

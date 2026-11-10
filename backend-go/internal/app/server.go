package app

import (
	"context"
	"net/http"
	"time"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	httprouter "digidocs-mgt/backend-go/internal/transport/http/router"
)

func NewServer(cfg config.Config) (*http.Server, error) {
	container, err := bootstrap.BuildContainer(cfg)
	if err != nil {
		return nil, err
	}

	handler := httprouter.New(cfg, container)
	for _, job := range container.Background {
		go job(context.Background())
	}

	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}, nil
}

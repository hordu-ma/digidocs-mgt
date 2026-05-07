package main

import (
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/config"
)

type fakeServer struct {
	err error
}

func (s fakeServer) ListenAndServe() error {
	return s.err
}

func TestRunReturnsListenAndServeError(t *testing.T) {
	wantErr := errors.New("listen failed")
	err := run(
		func() config.Config {
			return config.Config{AppName: "test-api", HTTPAddr: ":0"}
		},
		func(cfg config.Config) (httpServer, error) {
			if cfg.AppName != "test-api" || cfg.HTTPAddr != ":0" {
				t.Fatalf("cfg = %#v, want loaded config passed to newServer", cfg)
			}
			return fakeServer{err: wantErr}, nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("run err=%v, want listen error", err)
	}
}

func TestRunReturnsNewServerError(t *testing.T) {
	wantErr := errors.New("build failed")
	err := run(
		func() config.Config {
			return config.Config{AppName: "test-api"}
		},
		func(config.Config) (httpServer, error) {
			return nil, wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("run err=%v, want newServer error", err)
	}
}

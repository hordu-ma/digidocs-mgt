package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type SystemHandler struct {
	cfg config.Config
}

func NewSystemHandler(cfg config.Config) SystemHandler {
	return SystemHandler{cfg: cfg}
}

func (h SystemHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.WriteData(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h SystemHandler) Info(w http.ResponseWriter, r *http.Request) {
	response.WriteData(w, http.StatusOK, map[string]string{
		"app_name": h.cfg.AppName,
		"env":      h.cfg.AppEnv,
	})
}

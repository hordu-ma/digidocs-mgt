package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Meta  any    `json:"meta,omitempty"`
	Code  string `json:"code,omitempty"`
	Error string `json:"message,omitempty"`
}

func WriteData(w http.ResponseWriter, statusCode int, data any) {
	writeJSON(w, statusCode, Envelope{Data: data})
}

func WriteWithMeta(w http.ResponseWriter, statusCode int, data any, meta any) {
	writeJSON(w, statusCode, Envelope{
		Data: data,
		Meta: meta,
	})
}

func WriteError(w http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(w, statusCode, Envelope{
		Code:  code,
		Error: message,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"code":"internal_error","message":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

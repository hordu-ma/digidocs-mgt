package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteData(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteData(rec, http.StatusCreated, map[string]string{"id": "1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	data := envelope.Data.(map[string]any)
	if data["id"] != "1" {
		t.Fatalf("unexpected data: %+v", envelope.Data)
	}
}

func TestWriteWithMeta(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteWithMeta(rec, http.StatusOK, []string{"a"}, map[string]int{"total": 1})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", rec.Header().Get("Content-Type"))
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["meta"] == nil {
		t.Fatalf("expected meta: %+v", body)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusForbidden, "forbidden", "permission denied")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != "forbidden" || envelope.Error != "permission denied" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

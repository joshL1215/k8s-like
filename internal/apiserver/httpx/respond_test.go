package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q want %q", ct, "application/json")
	}

	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("body key: got %q want %q", got["key"], "value")
	}
}

func TestWriteJSON_CustomStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusCreated, struct{ Name string }{"pod-1"})

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d want %d", w.Code, http.StatusCreated)
	}
}

func TestWriteErr_WithError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteErr(w, http.StatusBadRequest, "invalid input", fmt.Errorf("name is required"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", w.Code, http.StatusBadRequest)
	}

	var got map[string]string
	json.NewDecoder(w.Body).Decode(&got)
	if got["error"] != "invalid input" {
		t.Errorf("error: got %q want %q", got["error"], "invalid input")
	}
	if got["detail"] != "name is required" {
		t.Errorf("detail: got %q want %q", got["detail"], "name is required")
	}
}

func TestWriteErr_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteErr(w, http.StatusNotFound, "not found", nil)

	var got map[string]string
	json.NewDecoder(w.Body).Decode(&got)
	if got["detail"] != "" {
		t.Errorf("expected empty detail, got %q", got["detail"])
	}
}

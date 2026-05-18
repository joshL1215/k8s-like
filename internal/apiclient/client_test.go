package apiclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	return &Client{baseURL: srv.URL, http: srv.Client()}
}

func TestDoRequest_200OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"ok"`))
	}))
	defer srv.Close()

	resp, err := newTestClient(srv).doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDoRequest_201Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	resp, err := newTestClient(srv).doRequest(context.Background(), http.MethodPost, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDoRequest_ErrorStatusReturnsError(t *testing.T) {
	for _, code := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))
		_, err := newTestClient(srv).doRequest(context.Background(), http.MethodGet, "/test", nil)
		if err == nil {
			t.Errorf("expected error for status %d, got nil", code)
		}
		srv.Close()
	}
}

func TestDoRequest_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newTestClient(srv).doRequest(ctx, http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestGet_UsesGetMethod(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := newTestClient(srv).get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if got != http.MethodGet {
		t.Errorf("method: got %q want %q", got, http.MethodGet)
	}
}

func TestDelete_UsesDeleteMethod(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := newTestClient(srv).delete(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if got != http.MethodDelete {
		t.Errorf("method: got %q want %q", got, http.MethodDelete)
	}
}

func TestSendJSON_WritesBodyAndContentType(t *testing.T) {
	var body, ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := newTestClient(srv).sendJSON(context.Background(), http.MethodPost, "/test", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q want %q", ct, "application/json")
	}
	if !strings.Contains(body, "val") {
		t.Errorf("body missing expected content: %s", body)
	}
}

func TestDecode_DecodesJSONIntoType(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`{"name":"test"}`))
	resp := &http.Response{Body: body}

	got, err := decode[map[string]string](resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("name: got %q want %q", got["name"], "test")
	}
}

func TestDecode_InvalidJSONReturnsError(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`not json`))
	resp := &http.Response{Body: body}

	_, err := decode[map[string]string](resp)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDrain_ClosesBody(t *testing.T) {
	closed := false
	body := &closeTracker{
		Reader:   strings.NewReader(`{"data":"ignored"}`),
		onClose:  func() { closed = true },
	}
	drain(&http.Response{Body: body})
	if !closed {
		t.Error("expected body to be closed after drain")
	}
}

type closeTracker struct {
	io.Reader
	onClose func()
}

func (c *closeTracker) Close() error {
	c.onClose()
	return nil
}

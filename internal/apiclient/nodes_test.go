package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestGetNode(t *testing.T) {
	want := &corev1.Node{Name: "node-1", Address: "10.0.0.1", Status: corev1.NodeReady}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %q want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/nodes/node-1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).GetNode(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Address != want.Address {
		t.Errorf("address: got %q want %q", got.Address, want.Address)
	}
}

func TestCreateNode(t *testing.T) {
	input := &corev1.Node{Name: "node-1", Address: "10.0.0.1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %q want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/nodes" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(input)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).CreateNode(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != input.Name {
		t.Errorf("name: got %q want %q", got.Name, input.Name)
	}
}

func TestUpdateNode(t *testing.T) {
	input := &corev1.Node{Name: "node-1", Address: "10.0.0.1", Status: corev1.NodeNotReady}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: got %q want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/nodes/node-1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(input)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).UpdateNode(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != corev1.NodeNotReady {
		t.Errorf("status: got %q want %q", got.Status, corev1.NodeNotReady)
	}
}

func TestDeleteNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: got %q want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/nodes/node-1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	}))
	defer srv.Close()

	if err := newTestClient(srv).DeleteNode(context.Background(), "node-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListNodes(t *testing.T) {
	nodes := []*corev1.Node{
		{Name: "node-1", Address: "10.0.0.1"},
		{Name: "node-2", Address: "10.0.0.2"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(nodes)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).ListNodes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("count: got %d want 2", len(got))
	}
}

package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestGetPod(t *testing.T) {
	want := &corev1.Pod{Name: "p1", Namespace: "default", Status: corev1.PodRunning}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %q want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/namespace/default/pods/p1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).GetPod(context.Background(), "default", "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != want.Name || got.Status != want.Status {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestCreatePod(t *testing.T) {
	input := &corev1.Pod{Name: "p1", Namespace: "default", Image: "nginx"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %q want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/namespace/default/pods" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(input)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).CreatePod(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != input.Name {
		t.Errorf("name: got %q want %q", got.Name, input.Name)
	}
}

func TestUpdatePod(t *testing.T) {
	input := &corev1.Pod{Name: "p1", Namespace: "default", NodeName: "node-1", Status: corev1.PodScheduled}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method: got %q want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/namespace/default/pods/p1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(input)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).UpdatePod(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.NodeName != "node-1" {
		t.Errorf("nodeName: got %q want %q", got.NodeName, "node-1")
	}
}

func TestDeletePod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: got %q want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/namespace/default/pods/p1" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	}))
	defer srv.Close()

	if err := newTestClient(srv).DeletePod(context.Background(), "default", "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPods(t *testing.T) {
	pods := []*corev1.Pod{
		{Name: "p1", Namespace: "default"},
		{Name: "p2", Namespace: "default"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/namespace/default/pods" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(pods)
	}))
	defer srv.Close()

	got, err := newTestClient(srv).ListPods(context.Background(), "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("count: got %d want 2", len(got))
	}
}

func TestWatchPods_StreamsEvents(t *testing.T) {
	events := []corev1.WatchEvent{
		{EventType: corev1.AddEvent, ObjectType: "POD", Pod: &corev1.Pod{Name: "p1"}},
		{EventType: corev1.ModificationEvent, ObjectType: "POD", Pod: &corev1.Pod{Name: "p1", Status: corev1.PodRunning}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("watch") != "true" {
			t.Errorf("expected watch=true query param")
		}
		flusher := w.(http.Flusher)
		enc := json.NewEncoder(w)
		for _, ev := range events {
			enc.Encode(ev)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	ch, err := newTestClient(srv).WatchPods(context.Background(), "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range events {
		select {
		case got, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before all events received")
			}
			if got.EventType != want.EventType {
				t.Errorf("event type: got %q want %q", got.EventType, want.EventType)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	}
}

func TestWatchPods_NodeNameFilter_SetsQueryParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("nodeName") != "node-1" {
			t.Errorf("nodeName query: got %q want %q", r.URL.Query().Get("nodeName"), "node-1")
		}
		w.(http.Flusher).Flush()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	newTestClient(srv).WatchPods(ctx, "default", "node-1")
}

func TestWatchPods_ContextCancel_ClosesChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := newTestClient(srv).WatchPods(ctx, "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancel()

	select {
	case _, open := <-ch:
		if open {
			t.Error("expected channel to be closed after context cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for channel to close")
	}
}

package apiserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiserver"
	"github.com/joshL1215/k8s-like/internal/store"
)

type noopStore struct{}

func (noopStore) CreatePod(_ context.Context, p *corev1.Pod) (*corev1.Pod, error) {
	return p, nil
}
func (noopStore) GetPod(_ context.Context, _, _ string) (*corev1.Pod, error) {
	return nil, store.ErrPodNotExist
}
func (noopStore) UpdatePod(_ context.Context, p *corev1.Pod) (*corev1.Pod, error) { return p, nil }
func (noopStore) DeletePod(_ context.Context, _, _ string) error                   { return nil }
func (noopStore) ListPods(_ context.Context, _ string) ([]*corev1.Pod, error) {
	return []*corev1.Pod{}, nil
}
func (noopStore) CreateNode(_ context.Context, _ *corev1.Node) error        { return nil }
func (noopStore) GetNode(_ context.Context, _ string) (*corev1.Node, error) { return nil, store.ErrNodeNotExist }
func (noopStore) UpdateNode(_ context.Context, _ *corev1.Node) error        { return nil }
func (noopStore) DeleteNode(_ context.Context, _ string) error               { return nil }
func (noopStore) ListNodes(_ context.Context) ([]*corev1.Node, error)       { return []*corev1.Node{}, nil }

func TestCreateAPIServer_RegistersPodRoutes(t *testing.T) {
	srv := httptest.NewServer(apiserver.CreateAPIServer(noopStore{}).Handler())
	t.Cleanup(srv.Close)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/namespace/default/pods"},
		{"GET", "/api/v1/namespace/default/pods/my-pod"},
		{"POST", "/api/v1/namespace/default/pods"},
		{"PUT", "/api/v1/namespace/default/pods/my-pod"},
		{"DELETE", "/api/v1/namespace/default/pods/my-pod"},
	}

	for _, r := range routes {
		req, _ := http.NewRequest(r.method, srv.URL+r.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", r.method, r.path, err)
		}
		resp.Body.Close()
		// Mux returns text/plain 404 for unregistered routes; handler JSON 404 means route is registered.
		if resp.StatusCode == http.StatusNotFound && resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("%s %s: route not registered (mux 404)", r.method, r.path)
		}
	}
}

func TestCreateAPIServer_RegistersNodeRoutes(t *testing.T) {
	srv := httptest.NewServer(apiserver.CreateAPIServer(noopStore{}).Handler())
	t.Cleanup(srv.Close)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/nodes"},
		{"GET", "/api/v1/nodes/node-1"},
		{"POST", "/api/v1/nodes"},
		{"PUT", "/api/v1/nodes/node-1"},
		{"DELETE", "/api/v1/nodes/node-1"},
	}

	for _, r := range routes {
		req, _ := http.NewRequest(r.method, srv.URL+r.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", r.method, r.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound && resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("%s %s: route not registered (mux 404)", r.method, r.path)
		}
	}
}

package nodes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiserver/nodes"
	"github.com/joshL1215/k8s-like/internal/store"
)

type fakeStore struct {
	mu   sync.Mutex
	data map[string]*corev1.Node
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string]*corev1.Node)}
}

func (s *fakeStore) CreateNode(_ context.Context, n *corev1.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[n.Name]; ok {
		return store.ErrNodeExists
	}
	cp := *n
	s.data[n.Name] = &cp
	return nil
}

func (s *fakeStore) GetNode(_ context.Context, name string) (*corev1.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.data[name]
	if !ok {
		return nil, store.ErrNodeNotExist
	}
	out := *n
	return &out, nil
}

func (s *fakeStore) UpdateNode(_ context.Context, n *corev1.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[n.Name]; !ok {
		return store.ErrNodeNotExist
	}
	cp := *n
	s.data[n.Name] = &cp
	return nil
}

func (s *fakeStore) DeleteNode(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[name]; !ok {
		return store.ErrNodeNotExist
	}
	delete(s.data, name)
	return nil
}

func (s *fakeStore) ListNodes(_ context.Context) ([]*corev1.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*corev1.Node, 0, len(s.data))
	for _, n := range s.data {
		cp := *n
		out = append(out, &cp)
	}
	return out, nil
}

func (s *fakeStore) CreatePod(_ context.Context, _ *corev1.Pod) (*corev1.Pod, error) { return nil, nil }
func (s *fakeStore) GetPod(_ context.Context, _, _ string) (*corev1.Pod, error)       { return nil, nil }
func (s *fakeStore) UpdatePod(_ context.Context, _ *corev1.Pod) (*corev1.Pod, error)  { return nil, nil }
func (s *fakeStore) DeletePod(_ context.Context, _, _ string) error                   { return nil }
func (s *fakeStore) ListPods(_ context.Context, _ string) ([]*corev1.Pod, error)      { return nil, nil }

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	h := nodes.NewHandler(newFakeStore())
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func nodeURL(base, name string) string {
	if name == "" {
		return fmt.Sprintf("%s/api/v1/nodes", base)
	}
	return fmt.Sprintf("%s/api/v1/nodes/%s", base, name)
}

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeBody[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

// --- Create ---

func TestCreate_ReturnsCreatedNodeWithDefaultStatus(t *testing.T) {
	srv := newTestServer(t)
	input := corev1.Node{Name: "node-1", Address: "10.0.0.1"}

	resp := doJSON(t, "POST", nodeURL(srv.URL, ""), input)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	got := decodeBody[corev1.Node](t, resp)
	if got.Name != "node-1" {
		t.Errorf("name: got %q want %q", got.Name, "node-1")
	}
	if got.Status != corev1.NodeReady {
		t.Errorf("status: got %q want %q", got.Status, corev1.NodeReady)
	}
}

func TestCreate_MissingName_Returns400(t *testing.T) {
	srv := newTestServer(t)
	resp := doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Address: "10.0.0.1"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
	resp.Body.Close()
}

func TestCreate_Duplicate_Returns409(t *testing.T) {
	srv := newTestServer(t)
	node := corev1.Node{Name: "node-1", Address: "10.0.0.1"}

	doJSON(t, "POST", nodeURL(srv.URL, ""), node).Body.Close()
	resp := doJSON(t, "POST", nodeURL(srv.URL, ""), node)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusConflict)
	}
	resp.Body.Close()
}

// --- Get ---

func TestGet_ExistingNode_Returns200(t *testing.T) {
	srv := newTestServer(t)
	doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Name: "node-1", Address: "10.0.0.1"}).Body.Close()

	resp := doJSON(t, "GET", nodeURL(srv.URL, "node-1"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[corev1.Node](t, resp)
	if got.Name != "node-1" {
		t.Errorf("name: got %q want %q", got.Name, "node-1")
	}
}

func TestGet_MissingNode_Returns404(t *testing.T) {
	srv := newTestServer(t)
	resp := doJSON(t, "GET", nodeURL(srv.URL, "ghost"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

// --- Update ---

func TestUpdate_ExistingNode_Returns200(t *testing.T) {
	srv := newTestServer(t)
	doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Name: "node-1", Address: "10.0.0.1"}).Body.Close()

	updated := corev1.Node{Name: "node-1", Address: "10.0.0.1", Status: corev1.NodeNotReady}
	resp := doJSON(t, "PUT", nodeURL(srv.URL, "node-1"), updated)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[corev1.Node](t, resp)
	if got.Status != corev1.NodeNotReady {
		t.Errorf("status: got %q want %q", got.Status, corev1.NodeNotReady)
	}
}

func TestUpdate_MissingNode_Returns500(t *testing.T) {
	srv := newTestServer(t)
	resp := doJSON(t, "PUT", nodeURL(srv.URL, "ghost"), corev1.Node{Name: "ghost"})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	resp.Body.Close()
}

// --- Delete ---

func TestDelete_ExistingNode_Returns200(t *testing.T) {
	srv := newTestServer(t)
	doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Name: "node-1", Address: "10.0.0.1"}).Body.Close()

	resp := doJSON(t, "DELETE", nodeURL(srv.URL, "node-1"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	resp2 := doJSON(t, "GET", nodeURL(srv.URL, "node-1"), nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()
}

func TestDelete_MissingNode_Returns404(t *testing.T) {
	srv := newTestServer(t)
	resp := doJSON(t, "DELETE", nodeURL(srv.URL, "ghost"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

// --- List ---

func TestList_ReturnsAllNodes(t *testing.T) {
	srv := newTestServer(t)
	doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Name: "node-1", Address: "10.0.0.1"}).Body.Close()
	doJSON(t, "POST", nodeURL(srv.URL, ""), corev1.Node{Name: "node-2", Address: "10.0.0.2"}).Body.Close()

	resp := doJSON(t, "GET", nodeURL(srv.URL, ""), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[[]*corev1.Node](t, resp)
	if len(got) != 2 {
		t.Errorf("count: got %d want 2", len(got))
	}
}

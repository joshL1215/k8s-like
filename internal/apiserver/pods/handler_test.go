package pods_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiserver/pods"
	"github.com/joshL1215/k8s-like/internal/apiserver/watchers"
	"github.com/joshL1215/k8s-like/internal/store"
)

// fakeStore is a thread-safe in-memory implementation of store.StoreInterface.
type fakeStore struct {
	mu   sync.Mutex
	data map[string]*corev1.Pod
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string]*corev1.Pod)}
}

func podKey(ns, name string) string { return ns + "/" + name }

func (s *fakeStore) CreatePod(_ context.Context, p *corev1.Pod) (*corev1.Pod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := podKey(p.Namespace, p.Name)
	if _, ok := s.data[k]; ok {
		return nil, store.ErrPodExists
	}
	cp := *p
	s.data[k] = &cp
	out := cp
	return &out, nil
}

func (s *fakeStore) GetPod(_ context.Context, ns, name string) (*corev1.Pod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data[podKey(ns, name)]
	if !ok {
		return nil, store.ErrPodNotExist
	}
	out := *p
	return &out, nil
}

func (s *fakeStore) UpdatePod(_ context.Context, p *corev1.Pod) (*corev1.Pod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := podKey(p.Namespace, p.Name)
	if _, ok := s.data[k]; !ok {
		return nil, store.ErrPodNotExist
	}
	cp := *p
	s.data[k] = &cp
	out := cp
	return &out, nil
}

func (s *fakeStore) DeletePod(_ context.Context, ns, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := podKey(ns, name)
	if _, ok := s.data[k]; !ok {
		return store.ErrPodNotExist
	}
	delete(s.data, k)
	return nil
}

func (s *fakeStore) ListPods(_ context.Context, ns string) ([]*corev1.Pod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := ns + "/"
	out := make([]*corev1.Pod, 0)
	for k, p := range s.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *fakeStore) CreateNode(_ context.Context, _ *corev1.Node) error          { return nil }
func (s *fakeStore) GetNode(_ context.Context, _ string) (*corev1.Node, error)   { return nil, nil }
func (s *fakeStore) UpdateNode(_ context.Context, _ *corev1.Node) error          { return nil }
func (s *fakeStore) DeleteNode(_ context.Context, _ string) error                 { return nil }
func (s *fakeStore) ListNodes(_ context.Context) ([]*corev1.Node, error)         { return nil, nil }

func newTestServer(t *testing.T) (*httptest.Server, *watchers.WatchManager) {
	t.Helper()
	fs := newFakeStore()
	wm := watchers.NewWatchManager()
	h := pods.NewHandler(fs, wm)
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, wm
}

func podURL(base, ns, name string) string {
	if name == "" {
		return fmt.Sprintf("%s/api/v1/namespace/%s/pods", base, ns)
	}
	return fmt.Sprintf("%s/api/v1/namespace/%s/pods/%s", base, ns, name)
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

func TestCreate_ReturnsCreatedPodWithDefaults(t *testing.T) {
	srv, _ := newTestServer(t)
	input := corev1.Pod{Name: "p1", Image: "nginx"}

	resp := doJSON(t, "POST", podURL(srv.URL, "default", ""), input)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusCreated)
	}

	got := decodeBody[corev1.Pod](t, resp)
	if got.Name != "p1" {
		t.Errorf("name: got %q want %q", got.Name, "p1")
	}
	if got.Namespace != "default" {
		t.Errorf("namespace: got %q want %q", got.Namespace, "default")
	}
	if got.Status != corev1.PodPending {
		t.Errorf("status: got %q want %q", got.Status, corev1.PodPending)
	}
}

func TestCreate_MissingName_Returns400(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Image: "nginx"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
	resp.Body.Close()
}

func TestCreate_Duplicate_Returns409(t *testing.T) {
	srv, _ := newTestServer(t)
	pod := corev1.Pod{Name: "p1", Image: "nginx"}

	doJSON(t, "POST", podURL(srv.URL, "default", ""), pod).Body.Close()
	resp := doJSON(t, "POST", podURL(srv.URL, "default", ""), pod)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusConflict)
	}
	resp.Body.Close()
}

// --- Get ---

func TestGet_ExistingPod_Returns200(t *testing.T) {
	srv, _ := newTestServer(t)
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()

	resp := doJSON(t, "GET", podURL(srv.URL, "default", "p1"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[corev1.Pod](t, resp)
	if got.Name != "p1" {
		t.Errorf("name: got %q want %q", got.Name, "p1")
	}
}

func TestGet_MissingPod_Returns404(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, "GET", podURL(srv.URL, "default", "ghost"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

// --- Update ---

func TestUpdate_ExistingPod_Returns200(t *testing.T) {
	srv, _ := newTestServer(t)
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()

	updated := corev1.Pod{Name: "p1", Namespace: "default", Image: "nginx:2", Status: corev1.PodRunning}
	resp := doJSON(t, "PUT", podURL(srv.URL, "default", "p1"), updated)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[corev1.Pod](t, resp)
	if got.Image != "nginx:2" {
		t.Errorf("image: got %q want %q", got.Image, "nginx:2")
	}
}

func TestUpdate_MissingPod_Returns404(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, "PUT", podURL(srv.URL, "default", "ghost"), corev1.Pod{Name: "ghost", Namespace: "default"})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

// --- Delete ---

func TestDelete_ExistingPod_Returns200(t *testing.T) {
	srv, _ := newTestServer(t)
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()

	resp := doJSON(t, "DELETE", podURL(srv.URL, "default", "p1"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	resp2 := doJSON(t, "GET", podURL(srv.URL, "default", "p1"), nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()
}

func TestDelete_MissingPod_Returns404(t *testing.T) {
	srv, _ := newTestServer(t)
	resp := doJSON(t, "DELETE", podURL(srv.URL, "default", "ghost"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
	resp.Body.Close()
}

// --- List ---

func TestList_ReturnsPodsinNamespace(t *testing.T) {
	srv, _ := newTestServer(t)
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p2", Image: "redis"}).Body.Close()

	resp := doJSON(t, "GET", podURL(srv.URL, "default", ""), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	got := decodeBody[[]*corev1.Pod](t, resp)
	if len(got) != 2 {
		t.Errorf("count: got %d want 2", len(got))
	}
}

func TestList_FilterByNodeName(t *testing.T) {
	srv, _ := newTestServer(t)
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()
	doJSON(t, "PUT", podURL(srv.URL, "default", "p1"), corev1.Pod{Name: "p1", Namespace: "default", NodeName: "node-1"}).Body.Close()
	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p2", Image: "redis"}).Body.Close()

	resp := doJSON(t, "GET", podURL(srv.URL, "default", "")+"?nodeName=node-1", nil)
	got := decodeBody[[]*corev1.Pod](t, resp)
	if len(got) != 1 || got[0].Name != "p1" {
		t.Errorf("expected only p1 with nodeName filter, got %+v", got)
	}
}

// --- Watch ---

func TestWatch_StreamsCreateEvent(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest("GET", podURL(srv.URL, "default", "")+"?watch=true", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open watch: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("watch status: got %d want %d", resp.StatusCode, http.StatusOK)
	}

	go func() {
		doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "w1", Image: "nginx"}).Body.Close()
	}()

	type result struct {
		ev  corev1.WatchEvent
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var ev corev1.WatchEvent
		err := json.NewDecoder(resp.Body).Decode(&ev)
		ch <- result{ev, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("decode event: %v", r.err)
		}
		if r.ev.EventType != corev1.AddEvent {
			t.Errorf("event type: got %q want %q", r.ev.EventType, corev1.AddEvent)
		}
		if r.ev.Pod == nil || r.ev.Pod.Name != "w1" {
			t.Errorf("pod name: got %v", r.ev.Pod)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for watch event")
	}
}

func TestWatch_NodeNameFilter_OnlyDeliverMatchingPods(t *testing.T) {
	srv, _ := newTestServer(t)

	req, _ := http.NewRequest("GET", podURL(srv.URL, "default", "")+"?watch=true&nodeName=node-1", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open watch: %v", err)
	}
	defer resp.Body.Close()

	go func() {
		// p-other has no nodeName — should be filtered
		doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p-other", Image: "nginx"}).Body.Close()
		// p-match has nodeName=node-1 — should arrive
		doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "p-match", Image: "nginx", NodeName: "node-1"}).Body.Close()
	}()

	ch := make(chan corev1.WatchEvent, 1)
	go func() {
		var ev corev1.WatchEvent
		json.NewDecoder(resp.Body).Decode(&ev)
		ch <- ev
	}()

	select {
	case ev := <-ch:
		if ev.Pod == nil || ev.Pod.Name != "p-match" {
			t.Errorf("expected p-match, got %v", ev.Pod)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for filtered watch event")
	}
}

func TestWatch_PublishesEventsOnCRUD(t *testing.T) {
	srv, wm := newTestServer(t)

	events := make(chan corev1.WatchEvent, 10)
	_, unsub := wm.Subscribe(func(e corev1.WatchEvent) bool { return true })
	defer unsub()
	_, unsub2 := wm.Subscribe(func(e corev1.WatchEvent) bool {
		events <- e
		return true
	})
	defer unsub2()

	doJSON(t, "POST", podURL(srv.URL, "default", ""), corev1.Pod{Name: "ev1", Image: "nginx"}).Body.Close()
	doJSON(t, "PUT", podURL(srv.URL, "default", "ev1"), corev1.Pod{Name: "ev1", Namespace: "default", Image: "nginx:2"}).Body.Close()
	doJSON(t, "DELETE", podURL(srv.URL, "default", "ev1"), nil).Body.Close()

	wantTypes := []corev1.EventType{corev1.AddEvent, corev1.ModificationEvent, corev1.DeletionEvent}
	for _, want := range wantTypes {
		select {
		case ev := <-events:
			if ev.EventType != want {
				t.Errorf("event type: got %q want %q", ev.EventType, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for %q event", want)
		}
	}
}

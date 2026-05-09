package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestResourcePath(t *testing.T) {
	cases := []struct {
		input      string
		wantPath   string
		wantNs     bool
		wantErrMsg string
	}{
		{"pod", "pods", true, ""},
		{"pods", "pods", true, ""},
		{"po", "pods", true, ""},
		{"Pod", "pods", true, ""},
		{"PODS", "pods", true, ""},
		{"node", "nodes", false, ""},
		{"nodes", "nodes", false, ""},
		{"no", "nodes", false, ""},
		{"unknown", "", false, `unknown resource "unknown"`},
	}

	for _, tc := range cases {
		path, ns, err := resourcePath(tc.input)
		if tc.wantErrMsg != "" {
			if err == nil || err.Error() != tc.wantErrMsg {
				t.Errorf("resourcePath(%q) error: got %v want %q", tc.input, err, tc.wantErrMsg)
			}
			continue
		}
		if err != nil {
			t.Errorf("resourcePath(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if path != tc.wantPath {
			t.Errorf("resourcePath(%q) path: got %q want %q", tc.input, path, tc.wantPath)
		}
		if ns != tc.wantNs {
			t.Errorf("resourcePath(%q) namespaced: got %v want %v", tc.input, ns, tc.wantNs)
		}
	}
}

func TestResourceURL(t *testing.T) {
	server = "http://localhost:5173"
	namespace = "kube-system"

	cases := []struct {
		kind    string
		name    string
		wantURL string
	}{
		{"pods", "", "http://localhost:5173/api/v1/namespace/kube-system/pods"},
		{"pods", "my-pod", "http://localhost:5173/api/v1/namespace/kube-system/pods/my-pod"},
		{"nodes", "", "http://localhost:5173/api/v1/nodes"},
		{"nodes", "node-1", "http://localhost:5173/api/v1/nodes/node-1"},
	}

	for _, tc := range cases {
		got, err := resourceURL(tc.kind, tc.name)
		if err != nil {
			t.Errorf("resourceURL(%q, %q) error: %v", tc.kind, tc.name, err)
			continue
		}
		if got != tc.wantURL {
			t.Errorf("resourceURL(%q, %q): got %q want %q", tc.kind, tc.name, got, tc.wantURL)
		}
	}
}

func TestDoRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"test"}`))
	}))
	defer srv.Close()

	data, err := doRequest("GET", srv.URL, nil)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("name: got %q want %q", got["name"], "test")
	}
}

func TestDoRequest_ParsedErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Pod not found","detail":"no such pod"}`))
	}))
	defer srv.Close()

	_, err := doRequest("GET", srv.URL, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if err.Error() != "Pod not found: no such pod" {
		t.Errorf("error message: got %q", err.Error())
	}
}

func TestDoRequest_WithBody(t *testing.T) {
	var received map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := doRequest("POST", srv.URL, map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if received["hello"] != "world" {
		t.Errorf("body: got %q want %q", received["hello"], "world")
	}
}

func TestDecodeManifest_Pod(t *testing.T) {
	data := []byte(`{"kind":"Pod","name":"my-pod","namespace":"default","image":"nginx"}`)
	kind, obj, err := decodeManifest(data)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if kind != "pods" {
		t.Errorf("kind: got %q want %q", kind, "pods")
	}
	p, ok := obj.(*corev1.Pod)
	if !ok {
		t.Fatalf("expected *corev1.Pod, got %T", obj)
	}
	if p.Name != "my-pod" {
		t.Errorf("pod name: got %q want %q", p.Name, "my-pod")
	}
}

func TestDecodeManifest_Node(t *testing.T) {
	data := []byte(`{"kind":"Node","name":"node-1","address":"10.0.0.1"}`)
	kind, obj, err := decodeManifest(data)
	if err != nil {
		t.Fatalf("decodeManifest: %v", err)
	}
	if kind != "nodes" {
		t.Errorf("kind: got %q want %q", kind, "nodes")
	}
	n, ok := obj.(*corev1.Node)
	if !ok {
		t.Fatalf("expected *corev1.Node, got %T", obj)
	}
	if n.Name != "node-1" {
		t.Errorf("node name: got %q want %q", n.Name, "node-1")
	}
}

func TestDecodeManifest_UnknownKind(t *testing.T) {
	data := []byte(`{"kind":"Deployment","name":"my-app"}`)
	_, _, err := decodeManifest(data)
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestDecodeManifest_InvalidJSON(t *testing.T) {
	_, _, err := decodeManifest([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

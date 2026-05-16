//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestNode_CreateAndGet(t *testing.T) {
	name := testNS(t)
	input := corev1.Node{Name: name, Address: "10.0.0.1"}

	resp := do(t, "POST", nodeURL(""), input)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	created := decode[corev1.Node](t, resp)
	if created.Status != corev1.NodeReady {
		t.Errorf("default status: got %q want %q", created.Status, corev1.NodeReady)
	}

	resp2 := do(t, "GET", nodeURL(name), nil)
	if resp2.StatusCode != http.StatusOK {
		resp2.Body.Close()
		t.Fatalf("get: got %d want %d", resp2.StatusCode, http.StatusOK)
	}
	got := decode[corev1.Node](t, resp2)
	if got.Address != "10.0.0.1" {
		t.Errorf("address: got %q want %q", got.Address, "10.0.0.1")
	}
}

func TestNode_CreateConflict(t *testing.T) {
	name := testNS(t)
	node := corev1.Node{Name: name, Address: "10.0.0.2"}

	do(t, "POST", nodeURL(""), node).Body.Close()

	resp := do(t, "POST", nodeURL(""), node)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate create: got %d want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestNode_UpdateStatus(t *testing.T) {
	name := testNS(t)
	do(t, "POST", nodeURL(""), corev1.Node{Name: name, Address: "10.0.0.3"}).Body.Close()

	updated := corev1.Node{Name: name, Address: "10.0.0.3", Status: corev1.NodeNotReady}
	resp := do(t, "PUT", nodeURL(name), updated)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("update: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	got := decode[corev1.Node](t, do(t, "GET", nodeURL(name), nil))
	if got.Status != corev1.NodeNotReady {
		t.Errorf("status: got %q want %q", got.Status, corev1.NodeNotReady)
	}
}

func TestNode_DeleteAndConfirmGone(t *testing.T) {
	name := testNS(t)
	do(t, "POST", nodeURL(""), corev1.Node{Name: name, Address: "10.0.0.4"}).Body.Close()

	resp := do(t, "DELETE", nodeURL(name), nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("delete: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	resp2 := do(t, "GET", nodeURL(name), nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: got %d want %d", resp2.StatusCode, http.StatusNotFound)
	}
}

func TestNode_GetMissing(t *testing.T) {
	resp := do(t, "GET", nodeURL("definitely-does-not-exist"), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
}

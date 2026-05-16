//go:build e2e

package e2e

import (
	"net/http"
	"sync"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestPod_CreateAndGet(t *testing.T) {
	ns := testNS(t)
	input := corev1.Pod{Name: "p1", Image: "nginx:latest"}

	resp := do(t, "POST", podURL(ns, ""), input)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	created := decode[corev1.Pod](t, resp)

	if created.Namespace != ns {
		t.Errorf("namespace: got %q want %q", created.Namespace, ns)
	}
	if created.Status != corev1.PodPending {
		t.Errorf("status: got %q want %q", created.Status, corev1.PodPending)
	}

	resp2 := do(t, "GET", podURL(ns, "p1"), nil)
	if resp2.StatusCode != http.StatusOK {
		resp2.Body.Close()
		t.Fatalf("get: got %d want %d", resp2.StatusCode, http.StatusOK)
	}
	got := decode[corev1.Pod](t, resp2)
	if got.Name != "p1" || got.Image != "nginx:latest" {
		t.Errorf("got %+v", got)
	}
}

func TestPod_CreateConflict_EtcdTxnEnforced(t *testing.T) {
	ns := testNS(t)
	pod := corev1.Pod{Name: "conflict", Image: "nginx"}

	resp := do(t, "POST", podURL(ns, ""), pod)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("first create: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := do(t, "POST", podURL(ns, ""), pod)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("duplicate create: got %d want %d", resp2.StatusCode, http.StatusConflict)
	}
}

func TestPod_ConcurrentCreate_OnlyOneSucceeds(t *testing.T) {
	ns := testNS(t)
	pod := corev1.Pod{Name: "race", Image: "nginx"}

	const goroutines = 10
	results := make([]int, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp := do(t, "POST", podURL(ns, ""), pod)
			results[idx] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	created := 0
	for _, code := range results {
		if code == http.StatusCreated {
			created++
		}
	}
	if created != 1 {
		t.Errorf("expected exactly 1 successful create, got %d (codes: %v)", created, results)
	}
}

func TestPod_UpdateAndGet(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "p1", Image: "nginx"}).Body.Close()

	updated := corev1.Pod{Name: "p1", Namespace: ns, Image: "nginx:2", Status: corev1.PodRunning, NodeName: "node-a"}
	resp := do(t, "PUT", podURL(ns, "p1"), updated)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("update: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	got := decode[corev1.Pod](t, do(t, "GET", podURL(ns, "p1"), nil))
	if got.Image != "nginx:2" {
		t.Errorf("image: got %q want %q", got.Image, "nginx:2")
	}
	if got.NodeName != "node-a" {
		t.Errorf("nodeName: got %q want %q", got.NodeName, "node-a")
	}
}

func TestPod_DeleteAndConfirmGone(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "del", Image: "nginx"}).Body.Close()

	resp := do(t, "DELETE", podURL(ns, "del"), nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("delete: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	resp2 := do(t, "GET", podURL(ns, "del"), nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: got %d want %d", resp2.StatusCode, http.StatusNotFound)
	}
}

func TestPod_DeleteAndRecreate(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "reborn", Image: "nginx"}).Body.Close()
	do(t, "DELETE", podURL(ns, "reborn"), nil).Body.Close()

	resp := do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "reborn", Image: "nginx:2"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("recreate: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestPod_ListInNamespace(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "a", Image: "nginx"}).Body.Close()
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "b", Image: "redis"}).Body.Close()

	resp := do(t, "GET", podURL(ns, ""), nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("list: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	pods := decode[[]*corev1.Pod](t, resp)
	if len(pods) != 2 {
		t.Errorf("count: got %d want 2", len(pods))
	}
}

func TestPod_ListFilterByNodeName(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "on-node", Image: "nginx", NodeName: "n1"}).Body.Close()
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "unscheduled", Image: "nginx"}).Body.Close()

	resp := do(t, "GET", podURL(ns, "")+"?nodeName=n1", nil)
	pods := decode[[]*corev1.Pod](t, resp)
	if len(pods) != 1 || pods[0].Name != "on-node" {
		t.Errorf("nodeName filter: got %+v", pods)
	}
}

func TestPod_NamespaceIsolation(t *testing.T) {
	ns1 := testNS(t) + "-ns1"
	ns2 := testNS(t) + "-ns2"

	do(t, "POST", podURL(ns1, ""), corev1.Pod{Name: "p", Image: "nginx"}).Body.Close()

	resp := do(t, "GET", podURL(ns2, ""), nil)
	pods := decode[[]*corev1.Pod](t, resp)
	if len(pods) != 0 {
		t.Errorf("ns2 should be empty, got %d pods", len(pods))
	}
}

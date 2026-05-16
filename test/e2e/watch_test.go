//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

// openWatch opens a streaming watch connection and returns the JSON decoder and
// a cleanup func. The connection is tied to the test lifetime.
func openWatch(t *testing.T, rawURL string) (*json.Decoder, func()) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := u.Query()
	q.Set("watch", "true")
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Fatalf("watch request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open watch: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("watch status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	return json.NewDecoder(resp.Body), func() { resp.Body.Close() }
}

// nextEvent reads the next WatchEvent from the stream, failing if it takes longer than d.
func nextEvent(t *testing.T, dec *json.Decoder, d time.Duration) corev1.WatchEvent {
	t.Helper()
	type result struct {
		ev  corev1.WatchEvent
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var ev corev1.WatchEvent
		err := dec.Decode(&ev)
		ch <- result{ev, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("decode watch event: %v", r.err)
		}
		return r.ev
	case <-time.After(d):
		t.Fatalf("timeout waiting for watch event after %s", d)
		return corev1.WatchEvent{}
	}
}

func TestWatch_ReceivesCreateEvent(t *testing.T) {
	ns := testNS(t)
	dec, close := openWatch(t, podURL(ns, ""))
	defer close()

	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "w1", Image: "nginx"}).Body.Close()

	ev := nextEvent(t, dec, 5*time.Second)
	if ev.EventType != corev1.AddEvent {
		t.Errorf("event type: got %q want %q", ev.EventType, corev1.AddEvent)
	}
	if ev.Pod == nil || ev.Pod.Name != "w1" {
		t.Errorf("pod: got %+v", ev.Pod)
	}
}

func TestWatch_ReceivesUpdateEvent(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "w2", Image: "nginx"}).Body.Close()

	dec, close := openWatch(t, podURL(ns, ""))
	defer close()

	do(t, "PUT", podURL(ns, "w2"), corev1.Pod{Name: "w2", Namespace: ns, Status: corev1.PodRunning}).Body.Close()

	ev := nextEvent(t, dec, 5*time.Second)
	if ev.EventType != corev1.ModificationEvent {
		t.Errorf("event type: got %q want %q", ev.EventType, corev1.ModificationEvent)
	}
	if ev.Pod == nil || ev.Pod.Status != corev1.PodRunning {
		t.Errorf("pod status: got %+v", ev.Pod)
	}
}

func TestWatch_ReceivesDeleteEvent(t *testing.T) {
	ns := testNS(t)
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "w3", Image: "nginx"}).Body.Close()

	dec, close := openWatch(t, podURL(ns, ""))
	defer close()

	do(t, "DELETE", podURL(ns, "w3"), nil).Body.Close()

	ev := nextEvent(t, dec, 5*time.Second)
	if ev.EventType != corev1.DeletionEvent {
		t.Errorf("event type: got %q want %q", ev.EventType, corev1.DeletionEvent)
	}
}

func TestWatch_NodeNameFilter(t *testing.T) {
	ns := testNS(t)
	dec, close := openWatch(t, podURL(ns, "")+"?nodeName=target-node")
	defer close()

	// This pod should be filtered out.
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "ignored", Image: "nginx"}).Body.Close()
	// This pod should arrive.
	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "matched", Image: "nginx", NodeName: "target-node"}).Body.Close()

	ev := nextEvent(t, dec, 5*time.Second)
	if ev.Pod == nil || ev.Pod.Name != "matched" {
		t.Errorf("expected matched, got %+v", ev.Pod)
	}
}

func TestWatch_MultipleSubscribersAllReceive(t *testing.T) {
	ns := testNS(t)
	dec1, close1 := openWatch(t, podURL(ns, ""))
	dec2, close2 := openWatch(t, podURL(ns, ""))
	defer close1()
	defer close2()

	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "fanout", Image: "nginx"}).Body.Close()

	for i, dec := range []*json.Decoder{dec1, dec2} {
		ev := nextEvent(t, dec, 5*time.Second)
		if ev.Pod == nil || ev.Pod.Name != "fanout" {
			t.Errorf("subscriber %d: got %+v", i, ev.Pod)
		}
	}
}

func TestWatch_FullLifecycle(t *testing.T) {
	ns := testNS(t)
	dec, close := openWatch(t, podURL(ns, ""))
	defer close()

	do(t, "POST", podURL(ns, ""), corev1.Pod{Name: "life", Image: "nginx"}).Body.Close()
	do(t, "PUT", podURL(ns, "life"), corev1.Pod{Name: "life", Namespace: ns, Status: corev1.PodRunning}).Body.Close()
	do(t, "DELETE", podURL(ns, "life"), nil).Body.Close()

	want := []corev1.EventType{corev1.AddEvent, corev1.ModificationEvent, corev1.DeletionEvent}
	for _, wantType := range want {
		ev := nextEvent(t, dec, 5*time.Second)
		if ev.EventType != wantType {
			t.Errorf("event type: got %q want %q", ev.EventType, wantType)
		}
	}
}

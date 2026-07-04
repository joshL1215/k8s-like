package kubelet

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type fakeClient struct {
	nodes     map[string]*corev1.Node
	pods      []*corev1.Pod
	events    chan corev1.WatchEvent
	watchNode string
	updates   []*corev1.Pod
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		nodes:  make(map[string]*corev1.Node),
		events: make(chan corev1.WatchEvent, 8),
	}
}

func (f *fakeClient) CreateNode(_ context.Context, node *corev1.Node) (*corev1.Node, error) {
	if _, exists := f.nodes[node.Name]; exists {
		return nil, errors.New("node exists")
	}
	f.nodes[node.Name] = cloneNode(node)
	return cloneNode(node), nil
}

func (f *fakeClient) GetNode(_ context.Context, name string) (*corev1.Node, error) {
	node, ok := f.nodes[name]
	if !ok {
		return nil, errors.New("node not found")
	}
	return cloneNode(node), nil
}

func (f *fakeClient) UpdateNode(_ context.Context, node *corev1.Node) (*corev1.Node, error) {
	f.nodes[node.Name] = cloneNode(node)
	return cloneNode(node), nil
}

func (f *fakeClient) ListPods(context.Context, string) ([]*corev1.Pod, error) {
	pods := make([]*corev1.Pod, 0, len(f.pods))
	for _, pod := range f.pods {
		pods = append(pods, clonePod(pod))
	}
	return pods, nil
}

func (f *fakeClient) WatchPods(_ context.Context, _ string, nodeName string) (<-chan corev1.WatchEvent, error) {
	f.watchNode = nodeName
	return f.events, nil
}

func (f *fakeClient) UpdatePod(_ context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	f.updates = append(f.updates, clonePod(pod))
	return clonePod(pod), nil
}

type recordingRuntime struct {
	started []string
	stopped []string
	status  corev1.PodStatus
}

func (r *recordingRuntime) StartPod(_ context.Context, pod *corev1.Pod) error {
	r.started = append(r.started, podKey(pod))
	return nil
}

func (r *recordingRuntime) StopPod(_ context.Context, pod *corev1.Pod) error {
	r.stopped = append(r.stopped, podKey(pod))
	return nil
}

func (r *recordingRuntime) PodStatus(context.Context, *corev1.Pod) (corev1.PodStatus, error) {
	if r.status == "" {
		return corev1.PodRunning, nil
	}
	return r.status, nil
}

func TestRunRegistersNodeSeedsPodsAndWatchesAssignedPods(t *testing.T) {
	client := newFakeClient()
	client.pods = []*corev1.Pod{
		{Name: "p1", Namespace: "default", NodeName: "node-1", Status: corev1.PodScheduled},
		{Name: "other", Namespace: "default", NodeName: "node-2", Status: corev1.PodScheduled},
	}
	runtime := &recordingRuntime{}
	k := New(client, &corev1.Node{Name: "node-1", Address: "127.0.0.1"}, runtime, "default")
	k.SetHeartbeatInterval(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- k.Run(ctx) }()

	waitFor(t, func() bool { return client.watchNode == "node-1" })
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := client.nodes["node-1"]; got == nil || got.Status != corev1.NodeReady {
		t.Fatalf("node registration: got %+v", got)
	}
	if len(runtime.started) != 1 || runtime.started[0] != "default/p1" {
		t.Fatalf("started pods: got %v want [default/p1]", runtime.started)
	}
	if len(client.updates) != 1 || client.updates[0].Status != corev1.PodRunning {
		t.Fatalf("pod updates: got %+v", client.updates)
	}
}

func TestRunReconcilesWatchEvents(t *testing.T) {
	client := newFakeClient()
	runtime := &recordingRuntime{}
	k := New(client, &corev1.Node{Name: "node-1"}, runtime, "default")
	k.SetHeartbeatInterval(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- k.Run(ctx) }()

	waitFor(t, func() bool { return client.watchNode == "node-1" })
	client.events <- corev1.WatchEvent{
		EventType:  corev1.AddEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: "p1", Namespace: "default", NodeName: "node-1", Status: corev1.PodScheduled},
	}
	waitFor(t, func() bool { return len(client.updates) == 1 })

	client.events <- corev1.WatchEvent{
		EventType:  corev1.DeletionEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: "p1", Namespace: "default", NodeName: "node-1"},
	}
	waitFor(t, func() bool { return len(runtime.stopped) == 1 })

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if runtime.stopped[0] != "default/p1" {
		t.Fatalf("stopped pods: got %v", runtime.stopped)
	}
}

func TestRegisterNodeUpdatesExistingNode(t *testing.T) {
	client := newFakeClient()
	client.nodes["node-1"] = &corev1.Node{Name: "node-1", Address: "old", Status: corev1.NodeNotReady}
	k := New(client, &corev1.Node{Name: "node-1", Address: "new"}, &recordingRuntime{}, "default")

	if err := k.registerNode(context.Background()); err != nil {
		t.Fatalf("registerNode: %v", err)
	}
	if got := client.nodes["node-1"].Address; got != "new" {
		t.Fatalf("address: got %q want new", got)
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

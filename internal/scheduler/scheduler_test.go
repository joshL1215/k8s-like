package scheduler

import (
	"context"
	"errors"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type fakePodWatcher struct {
	events    chan corev1.WatchEvent
	called    chan struct{}
	namespace string
	nodeName  string
	err       error
}

func newFakePodWatcher() *fakePodWatcher {
	return &fakePodWatcher{
		events: make(chan corev1.WatchEvent, 8),
		called: make(chan struct{}, 1),
	}
}

func (f *fakePodWatcher) WatchPods(ctx context.Context, namespace, nodeName string) (<-chan corev1.WatchEvent, error) {
	f.namespace = namespace
	f.nodeName = nodeName
	f.called <- struct{}{}
	return f.events, f.err
}

func (f *fakePodWatcher) ListNodes(context.Context) ([]*corev1.Node, error) {
	return nil, nil
}

func (f *fakePodWatcher) UpdatePod(context.Context, *corev1.Pod) (*corev1.Pod, error) {
	return nil, nil
}

func TestScheduler_RunWatchesAPIServerPods(t *testing.T) {
	watcher := newFakePodWatcher()
	s := New(watcher, "workloads")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx)
	}()

	<-watcher.called
	if watcher.namespace != "workloads" {
		t.Errorf("namespace: got %q want workloads", watcher.namespace)
	}
	if watcher.nodeName != "" {
		t.Errorf("nodeName: got %q want empty", watcher.nodeName)
	}

	watcher.events <- corev1.WatchEvent{
		EventType:  corev1.AddEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: "p1", Namespace: "workloads", Status: corev1.PodPending},
	}

	close(watcher.events)
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestScheduler_RunIgnoresScheduledAndDeletedPods(t *testing.T) {
	watcher := newFakePodWatcher()
	s := New(watcher, "default")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(context.Background())
	}()
	<-watcher.called

	watcher.events <- corev1.WatchEvent{
		EventType:  corev1.AddEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: "scheduled", Namespace: "default", NodeName: "node-1", Status: corev1.PodScheduled},
	}
	watcher.events <- corev1.WatchEvent{
		EventType:  corev1.DeletionEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: "deleted", Namespace: "default", Status: corev1.PodPending},
	}
	close(watcher.events)

	if err := <-errCh; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestScheduler_RunReturnsWatchErrors(t *testing.T) {
	watcher := newFakePodWatcher()
	watcher.err = errors.New("boom")

	err := New(watcher, "default").Run(context.Background())
	if err == nil {
		t.Fatal("expected watch error")
	}
}

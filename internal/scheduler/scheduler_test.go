package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

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

	pod, err := s.NextPod(withTimeout(t))
	if err != nil {
		t.Fatalf("next pod: %v", err)
	}
	if pod.Name != "p1" {
		t.Errorf("pod name: got %q want p1", pod.Name)
	}

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
	if s.QueueLen() != 0 {
		t.Errorf("queue len: got %d want 0", s.QueueLen())
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

func withTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	return ctx
}

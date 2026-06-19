package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestPriorityQueue_EnqueuePop(t *testing.T) {
	q := NewPriorityQueue()

	if !q.Enqueue(&corev1.Pod{Name: "p1", Namespace: "default"}) {
		t.Fatal("expected enqueue to add pod")
	}
	time.Sleep(time.Millisecond)
	if !q.Enqueue(&corev1.Pod{Name: "p2", Namespace: "default"}) {
		t.Fatal("expected enqueue to add second pod")
	}

	if q.Len() != 2 {
		t.Fatalf("len: got %d want 2", q.Len())
	}

	got, err := q.Pop(context.Background())
	if err != nil {
		t.Fatalf("pop: %v", err)
	}
	if got.Name != "p1" {
		t.Errorf("first pod: got %q want p1", got.Name)
	}

	got, err = q.Pop(context.Background())
	if err != nil {
		t.Fatalf("pop second: %v", err)
	}
	if got.Name != "p2" {
		t.Errorf("second pod: got %q want p2", got.Name)
	}
}

func TestPriorityQueue_EnqueueDeduplicatesActivePods(t *testing.T) {
	q := NewPriorityQueue()
	pod := &corev1.Pod{Name: "p1", Namespace: "default"}

	if !q.Enqueue(pod) {
		t.Fatal("expected first enqueue to add pod")
	}
	if q.Enqueue(pod) {
		t.Fatal("expected duplicate enqueue to be ignored")
	}
	if q.Len() != 1 {
		t.Errorf("len: got %d want 1", q.Len())
	}
}

func TestPriorityQueue_PopReturnsCopy(t *testing.T) {
	q := NewPriorityQueue()
	pod := &corev1.Pod{Name: "p1", Namespace: "default", Status: corev1.PodPending}
	q.Enqueue(pod)

	pod.Status = corev1.PodRunning

	got, err := q.Pop(context.Background())
	if err != nil {
		t.Fatalf("pop: %v", err)
	}
	if got.Status != corev1.PodPending {
		t.Errorf("status: got %q want %q", got.Status, corev1.PodPending)
	}
}

func TestPriorityQueue_PopHonorsContextCancel(t *testing.T) {
	q := NewPriorityQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := q.Pop(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pop error: got %v want context canceled", err)
	}
}

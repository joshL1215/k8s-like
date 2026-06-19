package queue

import (
	"container/heap"
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestActiveHeapOrdersByPriorityThenPushTimeThenKey(t *testing.T) {
	baseTime := time.Now()
	h := &ActiveHeap{}
	heap.Init(h)

	items := []*podHeapItem{
		{
			key:      "default/older-low",
			pod:      corev1.Pod{Name: "older-low", Namespace: "default"},
			priority: 1,
			pushedAt: baseTime,
		},
		{
			key:      "default/newer-high",
			pod:      corev1.Pod{Name: "newer-high", Namespace: "default"},
			priority: 10,
			pushedAt: baseTime.Add(time.Second),
		},
		{
			key:      "default/a-same-time",
			pod:      corev1.Pod{Name: "a-same-time", Namespace: "default"},
			priority: 10,
			pushedAt: baseTime.Add(2 * time.Second),
		},
		{
			key:      "default/b-same-time",
			pod:      corev1.Pod{Name: "b-same-time", Namespace: "default"},
			priority: 10,
			pushedAt: baseTime.Add(2 * time.Second),
		},
		{
			key:      "default/oldest-high",
			pod:      corev1.Pod{Name: "oldest-high", Namespace: "default"},
			priority: 10,
			pushedAt: baseTime.Add(-time.Second),
		},
	}
	for _, item := range items {
		heap.Push(h, item)
	}

	want := []string{
		"oldest-high",
		"newer-high",
		"a-same-time",
		"b-same-time",
		"older-low",
	}
	for _, wantName := range want {
		got := heap.Pop(h).(*podHeapItem)
		if got.pod.Name != wantName {
			t.Fatalf("pop order: got %q want %q", got.pod.Name, wantName)
		}
	}
}

func TestRetryHeapOrdersByRetryAt(t *testing.T) {
	baseTime := time.Now()
	h := &RetryHeap{}
	heap.Init(h)

	items := []*podHeapItem{
		{
			key:     "default/last",
			pod:     corev1.Pod{Name: "last", Namespace: "default"},
			retryAt: baseTime.Add(2 * time.Second),
		},
		{
			key:     "default/first",
			pod:     corev1.Pod{Name: "first", Namespace: "default"},
			retryAt: baseTime,
		},
		{
			key:     "default/second",
			pod:     corev1.Pod{Name: "second", Namespace: "default"},
			retryAt: baseTime.Add(time.Second),
		},
	}
	for _, item := range items {
		heap.Push(h, item)
	}

	want := []string{"first", "second", "last"}
	for _, wantName := range want {
		got := heap.Pop(h).(*podHeapItem)
		if got.pod.Name != wantName {
			t.Fatalf("pop order: got %q want %q", got.pod.Name, wantName)
		}
	}
}

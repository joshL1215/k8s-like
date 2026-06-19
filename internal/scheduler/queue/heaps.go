package queue

import (
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type podHeapItem struct {
	key      string
	pod      corev1.Pod
	priority int
	attempts int
	pushedAt time.Time
	retryAt  time.Time
}

type ActiveHeap []*podHeapItem

func (h ActiveHeap) Len() int      { return len(h) }
func (h ActiveHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *ActiveHeap) Push(x any)   { *h = append(*h, x.(*podHeapItem)) }

func (h ActiveHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	if !h[i].pushedAt.Equal(h[j].pushedAt) {
		return h[i].pushedAt.Before(h[j].pushedAt)
	}
	return h[i].key < h[j].key
}

func (h *ActiveHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return x
}

type RetryHeap []*podHeapItem

func (h RetryHeap) Len() int           { return len(h) }
func (h RetryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h RetryHeap) Less(i, j int) bool { return h[i].retryAt.Before(h[j].retryAt) }
func (h *RetryHeap) Push(x any)        { *h = append(*h, x.(*podHeapItem)) }

func (h *RetryHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return x
}

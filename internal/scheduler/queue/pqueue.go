package queue

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type PriorityQueue struct {
	mu         sync.Mutex
	cond       *sync.Cond
	activeHeap ActiveHeap
	retryHeap  RetryHeap
	queued     map[string]struct{}
}

func NewPriorityQueue() *PriorityQueue {
	q := &PriorityQueue{
		activeHeap: ActiveHeap{},
		retryHeap:  RetryHeap{},
		queued:     map[string]struct{}{},
	}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.activeHeap)
	heap.Init(&q.retryHeap)
	return q
}

func (q *PriorityQueue) Enqueue(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	key := podKey(pod)
	if _, exists := q.queued[key]; exists {
		return false
	}

	heap.Push(&q.activeHeap, &podHeapItem{
		key:      key,
		pod:      *pod,
		priority: 0,
		pushedAt: time.Now(),
	})
	q.queued[key] = struct{}{}
	q.cond.Signal()
	return true
}

func (q *PriorityQueue) Pop(ctx context.Context) (*corev1.Pod, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	stop := make(chan struct{})
	defer close(stop)
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				q.mu.Lock()
				defer q.mu.Unlock()
				q.cond.Broadcast()
			case <-stop:
			}
		}()
	}

	for q.activeHeap.Len() == 0 {
		if ctx != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		q.cond.Wait()
	}

	item := heap.Pop(&q.activeHeap).(*podHeapItem)
	delete(q.queued, item.key)

	pod := item.pod
	return &pod, nil
}

func (q *PriorityQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.activeHeap.Len()
}

func podKey(pod *corev1.Pod) string {
	return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
}

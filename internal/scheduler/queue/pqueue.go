package queue

import (
	"sync"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type PriorityQueue struct {
	mu         sync.Mutex
	activeHeap ActiveHeap
	retryHeap  RetryHeap
}

package scheduler

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiclient"
	"github.com/joshL1215/k8s-like/internal/scheduler/queue"
)

const defaultNamespace = "default"

type PodWatcher interface {
	WatchPods(ctx context.Context, namespace, nodeName string) (<-chan corev1.WatchEvent, error)
}

type Scheduler struct {
	client    PodWatcher
	namespace string
	queue     *queue.PriorityQueue
}

func NewForAPIServer(baseURL, namespace string) *Scheduler {
	return New(apiclient.NewClient(baseURL), namespace)
}

func New(client PodWatcher, namespace string) *Scheduler {
	if namespace == "" {
		namespace = defaultNamespace
	}
	return &Scheduler{
		client:    client,
		namespace: namespace,
		queue:     queue.NewPriorityQueue(),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("scheduler requires an api client")
	}

	events, err := s.client.WatchPods(ctx, s.namespace, "")
	if err != nil {
		return fmt.Errorf("watch pods: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if shouldEnqueue(event) {
				s.queue.Enqueue(event.Pod)
			}
		}
	}
}

func (s *Scheduler) NextPod(ctx context.Context) (*corev1.Pod, error) {
	if s == nil || s.queue == nil {
		return nil, errors.New("scheduler queue is not initialized")
	}
	return s.queue.Pop(ctx)
}

func (s *Scheduler) QueueLen() int {
	if s == nil || s.queue == nil {
		return 0
	}
	return s.queue.Len()
}

func shouldEnqueue(event corev1.WatchEvent) bool {
	if event.ObjectType != "POD" || event.EventType == corev1.DeletionEvent {
		return false
	}
	if event.Pod == nil || event.Pod.NodeName != "" || event.Pod.DeletionTimestamp != nil {
		return false
	}
	return event.Pod.Status == "" || event.Pod.Status == corev1.PodPending
}

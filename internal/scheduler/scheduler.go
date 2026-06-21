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

type SchedulerClient interface {
	WatchPods(ctx context.Context, namespace, nodeName string) (<-chan corev1.WatchEvent, error)
	ListNodes(ctx context.Context) ([]*corev1.Node, error)
	UpdatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error)
}

type Scheduler struct {
	client    SchedulerClient
	namespace string
	queue     *queue.PriorityQueue
}

func NewForAPIServer(baseURL, namespace string) *Scheduler {
	return New(apiclient.NewClient(baseURL), namespace)
}

func New(client SchedulerClient, namespace string) *Scheduler {
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
	return s.watchPods(ctx)
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

func (s *Scheduler) watchPods(ctx context.Context) error {
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

func (s *Scheduler) bind(ctx context.Context, pod *corev1.Pod, node *corev1.Node) error {
	if s == nil || s.client == nil {
		return errors.New("scheduler requires an api client")
	}

	if pod.Status != corev1.PodPending {
		return fmt.Errorf("%s is not pending, dropping pod", pod.Name)
	}

	pod.NodeName = node.Name
	pod.Status = corev1.PodScheduled
	_, err := s.client.UpdatePod(ctx, pod)
	return err
}

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
	// TODO: enqueue and schedule loop with policy
	return nil
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

func (s *Scheduler) bind(ctx context.Context, pod *corev1.Pod, node *corev1.Node) error {
	pod.NodeName = node.Name
	pod.Status = corev1.PodScheduled
	_, err := s.client.UpdatePod(ctx, pod)
	return err
}

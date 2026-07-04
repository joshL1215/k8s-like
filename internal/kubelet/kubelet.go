package kubelet

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiclient"
	"github.com/joshL1215/k8s-like/internal/kubelet/podcache"
)

const (
	defaultNamespace         = "default"
	defaultHeartbeatInterval = 10 * time.Second
)

type KubeletClient interface {
	CreateNode(ctx context.Context, node *corev1.Node) (*corev1.Node, error)
	GetNode(ctx context.Context, name string) (*corev1.Node, error)
	UpdateNode(ctx context.Context, node *corev1.Node) (*corev1.Node, error)
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)
	WatchPods(ctx context.Context, namespace, nodeName string) (<-chan corev1.WatchEvent, error)
	UpdatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error)
}

type Runtime interface {
	StartPod(ctx context.Context, pod *corev1.Pod) error
	StopPod(ctx context.Context, pod *corev1.Pod) error
	PodStatus(ctx context.Context, pod *corev1.Pod) (corev1.PodStatus, error)
}

type Kubelet struct {
	client            KubeletClient
	runtime           Runtime
	cache             *podcache.Cache
	node              *corev1.Node
	namespace         string
	heartbeatInterval time.Duration
}

func NewForAPIServer(baseURL string, node *corev1.Node, runtime Runtime, namespace string) *Kubelet {
	return New(apiclient.NewClient(baseURL), node, runtime, namespace)
}

func New(client KubeletClient, node *corev1.Node, runtime Runtime, namespace string) *Kubelet {
	if namespace == "" {
		namespace = defaultNamespace
	}
	return &Kubelet{
		client:            client,
		runtime:           runtime,
		cache:             podcache.NewCache(),
		node:              cloneNode(node),
		namespace:         namespace,
		heartbeatInterval: defaultHeartbeatInterval,
	}
}

func (k *Kubelet) Run(ctx context.Context) error {
	if err := k.validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := k.registerNode(ctx); err != nil {
		return err
	}
	if err := k.seedAssignedPods(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() { errCh <- k.heartbeat(ctx) }()
	go func() { errCh <- k.watchPods(ctx) }()

	err := <-errCh
	cancel()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (k *Kubelet) validate() error {
	if k == nil {
		return errors.New("kubelet is nil")
	}
	if k.client == nil {
		return errors.New("kubelet requires an api client")
	}
	if k.runtime == nil {
		return errors.New("kubelet requires a runtime")
	}
	if k.node == nil || k.node.Name == "" {
		return errors.New("kubelet requires a named node")
	}
	return nil
}

func (k *Kubelet) registerNode(ctx context.Context) error {
	node := cloneNode(k.node)
	node.Status = corev1.NodeReady
	node.Metrics = k.currentMetrics()

	created, err := k.client.CreateNode(ctx, node)
	if err == nil {
		k.node = cloneNode(created)
		return nil
	}

	existing, getErr := k.client.GetNode(ctx, node.Name)
	if getErr != nil {
		return fmt.Errorf("register node: create failed: %w; get existing failed: %w", err, getErr)
	}

	existing.Address = node.Address
	existing.AllowedLabels = cloneStringMap(node.AllowedLabels)
	existing.DisallowedLabels = cloneStringMap(node.DisallowedLabels)
	existing.Status = corev1.NodeReady
	existing.Metrics = node.Metrics

	updated, updateErr := k.client.UpdateNode(ctx, existing)
	if updateErr != nil {
		return fmt.Errorf("register node: update existing: %w", updateErr)
	}
	k.node = cloneNode(updated)
	return nil
}

func (k *Kubelet) seedAssignedPods(ctx context.Context) error {
	pods, err := k.client.ListPods(ctx, k.namespace)
	if err != nil {
		return fmt.Errorf("list pods: %w", err)
	}

	for _, pod := range pods {
		if pod == nil || pod.NodeName != k.node.Name {
			continue
		}
		if err := k.reconcilePod(ctx, pod); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kubelet) heartbeat(ctx context.Context) error {
	ticker := time.NewTicker(k.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := k.updateNodeStatus(ctx, corev1.NodeReady); err != nil {
				return err
			}
		}
	}
}

func (k *Kubelet) updateNodeStatus(ctx context.Context, status corev1.NodeStatus) error {
	node := cloneNode(k.node)
	node.Status = status

	node.Metrics = k.currentMetrics()

	updated, err := k.client.UpdateNode(ctx, node)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}
	k.node = cloneNode(updated)
	return nil
}

func (k *Kubelet) currentMetrics() *corev1.NodeMetrics {
	if k.node != nil && k.node.Metrics != nil {
		metrics := *k.node.Metrics
		metrics.PodCount = len(k.cache.List())
		return &metrics
	}
	return &corev1.NodeMetrics{PodCount: len(k.cache.List())}
}

func (k *Kubelet) watchPods(ctx context.Context) error {
	events, err := k.client.WatchPods(ctx, k.namespace, k.node.Name)
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
			if event.ObjectType != "POD" || event.Pod == nil {
				continue
			}
			if err := k.handlePodEvent(ctx, event); err != nil {
				return err
			}
		}
	}
}

func (k *Kubelet) handlePodEvent(ctx context.Context, event corev1.WatchEvent) error {
	switch event.EventType {
	case corev1.AddEvent, corev1.ModificationEvent:
		return k.reconcilePod(ctx, event.Pod)
	case corev1.DeletionEvent:
		return k.deletePod(ctx, event.Pod)
	default:
		return nil
	}
}

func (k *Kubelet) reconcilePod(ctx context.Context, pod *corev1.Pod) error {
	if pod == nil || pod.NodeName != k.node.Name {
		return nil
	}
	if pod.DeletionTimestamp != nil || pod.Status == corev1.PodTerminating || pod.Status == corev1.PodDeleted {
		return k.deletePod(ctx, pod)
	}

	desired := clonePod(pod)
	if desired.Status != corev1.PodRunning {
		if err := k.runtime.StartPod(ctx, desired); err != nil {
			return fmt.Errorf("start pod %s/%s: %w", desired.Namespace, desired.Name, err)
		}
		desired.Status = corev1.PodRunning
	}

	status, err := k.runtime.PodStatus(ctx, desired)
	if err != nil {
		return fmt.Errorf("pod status %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	if status != "" {
		desired.Status = status
	}

	k.cache.Set(desired)
	if pod.Status == desired.Status {
		return nil
	}
	if _, err := k.client.UpdatePod(ctx, desired); err != nil {
		return fmt.Errorf("update pod status %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	return nil
}

func (k *Kubelet) deletePod(ctx context.Context, pod *corev1.Pod) error {
	if pod == nil {
		return nil
	}
	if _, ok := k.cache.Get(pod.Namespace, pod.Name); !ok {
		return nil
	}
	if err := k.runtime.StopPod(ctx, pod); err != nil {
		return fmt.Errorf("stop pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	k.cache.Delete(pod.Namespace, pod.Name)
	return nil
}

func clonePod(pod *corev1.Pod) *corev1.Pod {
	if pod == nil {
		return nil
	}
	cp := *pod
	cp.Labels = cloneStringMap(pod.Labels)
	if pod.DeletionTimestamp != nil {
		ts := *pod.DeletionTimestamp
		cp.DeletionTimestamp = &ts
	}
	return &cp
}

func cloneNode(node *corev1.Node) *corev1.Node {
	if node == nil {
		return nil
	}
	cp := *node
	cp.AllowedLabels = cloneStringMap(node.AllowedLabels)
	cp.DisallowedLabels = cloneStringMap(node.DisallowedLabels)
	if node.Metrics != nil {
		metrics := *node.Metrics
		cp.Metrics = &metrics
	}
	return &cp
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

type FakeRuntime struct {
	mu      sync.Mutex
	running map[string]corev1.PodStatus
}

func NewFakeRuntime() *FakeRuntime {
	return &FakeRuntime{
		running: make(map[string]corev1.PodStatus),
	}
}

func (r *FakeRuntime) StartPod(_ context.Context, pod *corev1.Pod) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running == nil {
		r.running = make(map[string]corev1.PodStatus)
	}
	r.running[podKey(pod)] = corev1.PodRunning
	return nil
}

func (r *FakeRuntime) StopPod(_ context.Context, pod *corev1.Pod) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, podKey(pod))
	return nil
}

func (r *FakeRuntime) PodStatus(_ context.Context, pod *corev1.Pod) (corev1.PodStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[podKey(pod)], nil
}

func podKey(pod *corev1.Pod) string {
	if pod == nil {
		return "/"
	}
	return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
}

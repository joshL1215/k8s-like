package podcache

import (
	"fmt"
	"sync"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type Cache struct {
	mu   sync.RWMutex
	pods map[string]*corev1.Pod
}

func NewCache() *Cache {
	return &Cache{
		pods: make(map[string]*corev1.Pod),
	}
}

func (c *Cache) Set(pod *corev1.Pod) {
	if c == nil || pod == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.pods[podKey(pod.Namespace, pod.Name)] = clonePod(pod)
}

func (c *Cache) Get(namespace, name string) (*corev1.Pod, bool) {
	if c == nil {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	pod, ok := c.pods[podKey(namespace, name)]
	if !ok {
		return nil, false
	}
	return clonePod(pod), true
}

func (c *Cache) Delete(namespace, name string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pods, podKey(namespace, name))
}

func (c *Cache) List() []*corev1.Pod {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	pods := make([]*corev1.Pod, 0, len(c.pods))
	for _, pod := range c.pods {
		pods = append(pods, clonePod(pod))
	}
	return pods
}

func podKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func clonePod(pod *corev1.Pod) *corev1.Pod {
	if pod == nil {
		return nil
	}
	cp := *pod
	if pod.Labels != nil {
		cp.Labels = make(map[string]string, len(pod.Labels))
		for key, value := range pod.Labels {
			cp.Labels[key] = value
		}
	}
	if pod.DeletionTimestamp != nil {
		ts := *pod.DeletionTimestamp
		cp.DeletionTimestamp = &ts
	}
	return &cp
}

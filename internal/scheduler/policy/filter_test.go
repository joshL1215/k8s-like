package policy

import (
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestFilterNodesFiltersByAvailableResources(t *testing.T) {
	nodes := []*corev1.Node{
		node("fits", nil, nil, metrics(4, 1, 8<<30, 2<<30, 100<<30, 10<<30)),
		node("cpu-small", nil, nil, metrics(2, 1, 8<<30, 2<<30, 100<<30, 10<<30)),
		node("memory-small", nil, nil, metrics(4, 1, 4<<30, 3<<30, 100<<30, 10<<30)),
		node("disk-small", nil, nil, metrics(4, 1, 8<<30, 2<<30, 20<<30, 10<<30)),
		node("unknown", nil, nil, nil),
		nil,
	}
	pod := &corev1.Pod{
		Resources: corev1.Resources{CPU: 2, Memory: 2 << 30, Disk: 20 << 30},
	}

	got := FilterNodes(nodes, pod)
	if len(got) != 1 || got[0].Name != "fits" {
		t.Fatalf("filtered nodes: got %+v want only fits", names(got))
	}
}

func TestFilterNodesUsesDisallowedLabelsWhenAllowedLabelsEmpty(t *testing.T) {
	nodes := []*corev1.Node{
		node("general", nil, map[string]string{"workload": "batch"}, metrics(4, 0, 8<<30, 0, 100<<30, 0)),
		node("denied", nil, map[string]string{"workload": "frontend"}, metrics(4, 0, 8<<30, 0, 100<<30, 0)),
	}
	pod := &corev1.Pod{
		Labels:    map[string]string{"workload": "frontend"},
		Resources: corev1.Resources{CPU: 1, Memory: 1 << 30, Disk: 1 << 30},
	}

	got := FilterNodes(nodes, pod)
	if len(got) != 1 || got[0].Name != "general" {
		t.Fatalf("filtered nodes: got %+v want only general", names(got))
	}
}

func TestFilterNodesUsesAllowedLabelsWhenPresent(t *testing.T) {
	nodes := []*corev1.Node{
		node("allowed", map[string]string{"workload": "frontend"}, map[string]string{"workload": "frontend"}, metrics(4, 0, 8<<30, 0, 100<<30, 0)),
		node("not-allowed", map[string]string{"workload": "batch"}, nil, metrics(4, 0, 8<<30, 0, 100<<30, 0)),
		node("no-match", map[string]string{"tier": "api"}, nil, metrics(4, 0, 8<<30, 0, 100<<30, 0)),
	}
	pod := &corev1.Pod{
		Labels:    map[string]string{"workload": "frontend"},
		Resources: corev1.Resources{CPU: 1, Memory: 1 << 30, Disk: 1 << 30},
	}

	got := FilterNodes(nodes, pod)
	if len(got) != 1 || got[0].Name != "allowed" {
		t.Fatalf("filtered nodes: got %+v want only allowed", names(got))
	}
}

func TestFilterNodesNilPodReturnsNil(t *testing.T) {
	if got := FilterNodes([]*corev1.Node{node("n1", nil, nil, metrics(1, 0, 1, 0, 1, 0))}, nil); got != nil {
		t.Fatalf("filtered nodes: got %+v want nil", got)
	}
}

func node(name string, allowed, disallowed map[string]string, nodeMetrics *corev1.NodeMetrics) *corev1.Node {
	return &corev1.Node{
		Name:             name,
		AllowedLabels:    allowed,
		DisallowedLabels: disallowed,
		Metrics:          nodeMetrics,
	}
}

func metrics(cpuTotal, cpuUsed float64, memoryTotal, memoryUsed, diskTotal, diskUsed int64) *corev1.NodeMetrics {
	return &corev1.NodeMetrics{
		CPUTotalCores:    cpuTotal,
		CPUOccupiedCores: cpuUsed,
		MemoryTotalBytes: memoryTotal,
		MemoryUsedBytes:  memoryUsed,
		DiskTotalBytes:   diskTotal,
		DiskUsedBytes:    diskUsed,
	}
}

func names(nodes []*corev1.Node) []string {
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, node.Name)
	}
	return result
}

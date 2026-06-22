package policy

import corev1 "github.com/joshL1215/k8s-like/api/core/v1"

// FilterNodes returns the nodes that can run pod based on available resources
// and the node label allow/deny policy.
func FilterNodes(nodes []*corev1.Node, pod *corev1.Pod) []*corev1.Node {
	if pod == nil {
		return nil
	}

	filtered := make([]*corev1.Node, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if !fitsResources(node, pod.Resources) {
			continue
		}
		if !allowsPodLabels(node, pod.Labels) {
			continue
		}
		filtered = append(filtered, node)
	}

	return filtered
}

func fitsResources(node *corev1.Node, requested corev1.Resources) bool {
	if node.Metrics == nil {
		return false
	}

	return requested.CPU <= node.Metrics.CPUTotalCores-node.Metrics.CPUOccupiedCores &&
		requested.Memory <= node.Metrics.MemoryTotalBytes-node.Metrics.MemoryUsedBytes &&
		requested.Disk <= node.Metrics.DiskTotalBytes-node.Metrics.DiskUsedBytes
}

func allowsPodLabels(node *corev1.Node, podLabels map[string]string) bool {
	if len(node.AllowedLabels) > 0 {
		return matchesAnyLabel(podLabels, node.AllowedLabels)
	}
	return !matchesAnyLabel(podLabels, node.DisallowedLabels)
}

func matchesAnyLabel(podLabels, policyLabels map[string]string) bool {
	for key, value := range policyLabels {
		if podLabels[key] == value {
			return true
		}
	}
	return false
}

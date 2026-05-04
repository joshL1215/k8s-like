package watchers

import corev1 "github.com/joshL1215/k8s-like/api/core/v1"

type subscriber struct {
	ch     chan corev1.WatchEvent
	filter func(corev1.WatchEvent) bool
}

type WatchFilter func(corev1.WatchEvent) bool
type EventChannel <-chan corev1.WatchEvent

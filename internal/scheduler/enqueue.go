package scheduler

import (
	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

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

func shouldEnqueue(event corev1.WatchEvent) bool {
	if event.ObjectType != "POD" || event.EventType == corev1.DeletionEvent {
		return false
	}
	if event.Pod == nil || event.Pod.NodeName != "" || event.Pod.DeletionTimestamp != nil {
		return false
	}
	return event.Pod.Status == "" || event.Pod.Status == corev1.PodPending
}

package watchers

import (
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func podEvent(name string) corev1.WatchEvent {
	return corev1.WatchEvent{
		EventType:  corev1.AddEvent,
		ObjectType: "POD",
		Pod:        &corev1.Pod{Name: name, Namespace: "default"},
	}
}

func recvWithin(t *testing.T, ch EventChannel, d time.Duration) (corev1.WatchEvent, bool) {
	t.Helper()
	select {
	case e, ok := <-ch:
		return e, ok
	case <-time.After(d):
		return corev1.WatchEvent{}, false
	}
}

func TestSubscribe_ReceivesPublishedEvent(t *testing.T) {
	wm := NewWatchManager()
	ch, unsub := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	defer unsub()

	wm.Publish(podEvent("a"))

	got, ok := recvWithin(t, ch, time.Second)
	if !ok {
		t.Fatal("expected event, received nothing within timeout")
	}
	if got.Pod.Name != "a" {
		t.Errorf("pod name: got %q want %q", got.Pod.Name, "a")
	}
}

func TestSubscribe_FilterDropsNonMatchingEvents(t *testing.T) {
	wm := NewWatchManager()
	onlyB := func(e corev1.WatchEvent) bool { return e.Pod != nil && e.Pod.Name == "b" }
	ch, unsub := wm.Subscribe(onlyB)
	defer unsub()

	wm.Publish(podEvent("a"))
	if _, ok := recvWithin(t, ch, 50*time.Millisecond); ok {
		t.Fatal("filter should have dropped event for pod a")
	}

	wm.Publish(podEvent("b"))
	got, ok := recvWithin(t, ch, time.Second)
	if !ok || got.Pod.Name != "b" {
		t.Fatalf("expected pod b; ok=%v name=%q", ok, got.Pod.Name)
	}
}

func TestPublish_FanOutToMultipleSubscribers(t *testing.T) {
	wm := NewWatchManager()
	ch1, u1 := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	ch2, u2 := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	defer u1()
	defer u2()

	wm.Publish(podEvent("x"))

	for i, ch := range []EventChannel{ch1, ch2} {
		got, ok := recvWithin(t, ch, time.Second)
		if !ok || got.Pod.Name != "x" {
			t.Errorf("subscriber %d: ok=%v name=%q", i, ok, got.Pod.Name)
		}
	}
}

func TestUnsubscribe_ClosesChannel(t *testing.T) {
	wm := NewWatchManager()
	ch, unsub := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	unsub()

	_, ok := recvWithin(t, ch, time.Second)
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestPublish_SlowSubscriberDoesNotBlock(t *testing.T) {
	wm := NewWatchManager()
	_, unsub := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			wm.Publish(podEvent("a"))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}

func TestPublish_UnsubscribedDuringPublish_DoesNotPanic(t *testing.T) {
	wm := NewWatchManager()
	ch, unsub := wm.Subscribe(func(corev1.WatchEvent) bool { return true })
	_ = ch

	go func() {
		for i := 0; i < 100; i++ {
			wm.Publish(podEvent("a"))
		}
	}()
	unsub()
}

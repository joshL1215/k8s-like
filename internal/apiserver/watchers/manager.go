package watchers

import (
	"strconv"
	"sync"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

type WatchManager struct {
	subscribers map[string]*subscriber
	mutex       sync.RWMutex
	counter     int
}

func NewWatchManager() *WatchManager {
	return &WatchManager{
		subscribers: make(map[string]*subscriber),
	}
}

// return channel and cleanup function
func (wm *WatchManager) Subscribe(filter WatchFilter) (EventChannel, func()) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	id := strconv.Itoa(wm.counter)
	ch := make(chan corev1.WatchEvent, 64)
	wm.subscribers[id] = &subscriber{
		ch:     ch,
		filter: filter,
	}
	wm.counter += 1

	// := passes function local variable values
	unsubscribe := func() {
		wm.mutex.Lock()
		defer wm.mutex.Unlock()
		delete(wm.subscribers, id)
		close(ch)
	}
	return ch, unsubscribe
}

func (wm *WatchManager) Publish(event corev1.WatchEvent) {
	// using RLock since this only reads subscribers, faster
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	for _, subscriber := range wm.subscribers {
		if subscriber.filter(event) {
			select {
			case subscriber.ch <- event:
			default: // NOTE: silently drops, maybe implement drop slow reader or log
			}
		}
	}
}

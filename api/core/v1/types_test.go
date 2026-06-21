package v1

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestPod_JSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := Pod{
		Name:              "test-pod",
		Namespace:         "default",
		Image:             "nginx:latest",
		NodeName:          "node-1",
		Status:            PodRunning,
		DeletionTimestamp: &now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Pod
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name: got %q want %q", got.Name, original.Name)
	}
	if got.Status != original.Status {
		t.Errorf("Status: got %q want %q", got.Status, original.Status)
	}
	if got.DeletionTimestamp == nil || !got.DeletionTimestamp.Equal(now) {
		t.Errorf("DeletionTimestamp: got %v want %v", got.DeletionTimestamp, now)
	}
}

func TestPod_OmitsNodeNameWhenEmpty(t *testing.T) {
	p := Pod{Name: "p", Namespace: "default", Image: "nginx"}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)
	if _, ok := m["nodeName"]; ok {
		t.Error("expected nodeName to be omitted when empty")
	}
	if _, ok := m["deleteTime"]; ok {
		t.Error("expected deleteTime to be omitted when nil")
	}
}

func TestNode_JSONRoundTrip(t *testing.T) {
	original := Node{
		Name:    "node-1",
		Address: "10.0.0.1",
		Status:  NodeReady,
		Metrics: &NodeMetrics{
			CPUUsagePercent:  73.5,
			MemoryUsedBytes:  8 << 30,
			MemoryTotalBytes: 16 << 30,
			DiskUsedBytes:    120 << 30,
			DiskTotalBytes:   256 << 30,
			NetworkRxBytes:   1024,
			NetworkTxBytes:   2048,
			PodCount:         42,
			RunningProcesses: 314,
			LoadAverage1Min:  1.25,
			LoadAverage5Min:  1.10,
			LoadAverage15Min: 0.95,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Node
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Errorf("got %+v want %+v", got, original)
	}
}

func TestNode_OmitsMetricsWhenEmpty(t *testing.T) {
	node := Node{Name: "node-1", Address: "10.0.0.1", Status: NodeReady}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["metrics"]; ok {
		t.Error("expected metrics to be omitted when empty")
	}
}

func TestNodeMetrics_JSONRoundTrip(t *testing.T) {
	original := NodeMetrics{
		CPUUsagePercent:  87.2,
		MemoryUsedBytes:  12 << 30,
		MemoryTotalBytes: 32 << 30,
		DiskUsedBytes:    180 << 30,
		DiskTotalBytes:   512 << 30,
		NetworkRxBytes:   4096,
		NetworkTxBytes:   8192,
		PodCount:         128,
		RunningProcesses: 512,
		LoadAverage1Min:  3.5,
		LoadAverage5Min:  2.75,
		LoadAverage15Min: 1.95,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got NodeMetrics
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != original {
		t.Errorf("got %+v want %+v", got, original)
	}
}

func TestWatchEvent_JSONRoundTrip(t *testing.T) {
	pod := &Pod{Name: "p", Namespace: "default", Status: PodPending}
	original := WatchEvent{
		EventType:  AddEvent,
		ObjectType: "POD",
		Pod:        pod,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got WatchEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.EventType != AddEvent {
		t.Errorf("EventType: got %q want %q", got.EventType, AddEvent)
	}
	if got.Pod == nil || got.Pod.Name != "p" {
		t.Errorf("Pod: got %+v", got.Pod)
	}
	if got.Node != nil {
		t.Error("expected Node to be nil")
	}
}

func TestPodStatusConstants(t *testing.T) {
	statuses := []PodStatus{PodPending, PodScheduled, PodRunning, PodTerminating, PodDeleted}
	seen := map[PodStatus]bool{}
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate PodStatus: %q", s)
		}
		seen[s] = true
		if s == "" {
			t.Error("PodStatus must not be empty string")
		}
	}
}

func TestNodeStatusConstants(t *testing.T) {
	if NodeReady == NodeNotReady {
		t.Error("NodeReady and NodeNotReady must be distinct")
	}
}

package v1

import "time"

// how enums are done in Go
// Pod phase
type PodStatus string

const (
	PodPending     PodStatus = "Pending"
	PodScheduled   PodStatus = "Scheduled"
	PodRunning     PodStatus = "Running"
	PodTerminating PodStatus = "Terminating"
	PodDeleted     PodStatus = "Deleted"
)

type Pod struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Labels            map[string]string `json:"labels,omitempty"`
	Image             string            `json:"image"`
	Resources         Resources         `json:"resources,omitempty"`
	NodeName          string            `json:"nodeName,omitempty"`
	Status            PodStatus         `json:"phase"`
	DeletionTimestamp *time.Time        `json:"deleteTime,omitempty"`
}

type Resources struct {
	CPU    float64 `json:"cpu"`
	Memory int64   `json:"memory"`
	Disk   int64   `json:"disk"`
}

// Nodes
type NodeStatus string

const (
	NodeReady    NodeStatus = "Ready"
	NodeNotReady NodeStatus = "NotReady"
)

type NodeMetrics struct {
	CPUOccupiedCores float64 `json:"cpuOccupied"` // average used over some interval, can be measured as fractions of cores
	CPUTotalCores    float64 `json:"cpuAvailable"`
	MemoryUsedBytes  int64   `json:"memoryUsedBytes"`
	MemoryTotalBytes int64   `json:"memoryTotalBytes"`
	DiskUsedBytes    int64   `json:"diskUsedBytes"`
	DiskTotalBytes   int64   `json:"diskTotalBytes"`
	NetworkRxBytes   int64   `json:"networkRxBytes"`
	NetworkTxBytes   int64   `json:"networkTxBytes"`
	PodCount         int     `json:"podCount"`
	RunningProcesses int     `json:"runningProcesses"`
	LoadAverage1Min  float64 `json:"loadAverage1Min"`
	LoadAverage5Min  float64 `json:"loadAverage5Min"`
	LoadAverage15Min float64 `json:"loadAverage15Min"`
}

type Node struct {
	Name             string            `json:"name"`
	AllowedLabels    map[string]string `json:"allowedLabels,omitempty"`
	DisallowedLabels map[string]string `json:"disallowedLabels,omitempty"`
	Address          string            `json:"address"`
	Status           NodeStatus        `json:"status"`
	Metrics          *NodeMetrics      `json:"metrics,omitempty"`
}

// Events
type EventType string
type ObjectType string

const (
	AddEvent          EventType = "ADDED"
	ModificationEvent EventType = "MODIFIED"
	DeletionEvent     EventType = "DELETED"
)

type WatchEvent struct {
	EventType  EventType  `json:"eventType"`
	ObjectType ObjectType `json:"objectType"`
	Pod        *Pod       `json:"pod,omitempty"`
	Node       *Node      `json:"node,omitempty"`
}

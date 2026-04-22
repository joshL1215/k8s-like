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
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	Image             string     `json:"image"`
	NodeName          string     `json:"nodeName,omitempty"`
	Status            PodStatus  `json:"phase"`
	DeletionTimestamp *time.Time `json:"deleteTime,omitempty"`
}

// Nodes
type NodeStatus string

const (
	NodeReady    NodeStatus = "Ready"
	NodeNotReady NodeStatus = "NotReady"
)

type Node struct {
	Name    string     `json:"name"`
	Address string     `json:"address"`
	Status  NodeStatus `json:"status"`
}

// Events
type EventType string
type EventObject string

const (
	AddEvent          EventType = "ADDED"
	ModificationEvent EventType = "MODIFIED"
	DeletionEvent     EventType = "DELETED"
)

type WatchEvent struct {
	EventType   EventType   `json:"eventType"`
	EventObject EventObject `json:"objectType"`
	Pod         *Pod        `json:"pod,omitempty"`
	Node        *Node       `json:"node,omitempty"`
}

package v1

import "time"

// how enums are done in Go
// Pod phase
type PodPhase string

const (
	PodPending     PodPhase = "Pending"
	PodScheduled   PodPhase = "Scheduled"
	PodRunning     PodPhase = "Running"
	PodTerminating PodPhase = "Terminating"
	PodDeleted     PodPhase = "Deleted"
)

type Pod struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	Image             string     `json:"image"`
	NodeName          string     `json:"nodeName,omitempty"`
	Phase             PodPhase   `json:"phase"`
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

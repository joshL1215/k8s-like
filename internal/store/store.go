package store

import (
	"errors"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

var ErrNodeExists = errors.New("node already exists")
var ErrNodeNotExist = errors.New("node of this name does not exist")

var ErrPodExists = errors.New("pod already exists")
var ErrPodNotExist = errors.New("pod of this name does not exist")
var ErrPodIsDeleting = errors.New("pod is already being deleted")

// Defines an agnostic store interface
type StoreInterface interface {
	CreatePod(pod *corev1.Pod) error
	GetPod(namespace, name string) (*corev1.Pod, error)
	UpdatePod(pod *corev1.Pod) error
	DeletePod(namespace, name string) error
	ListPods(namespace string) ([]*corev1.Pod, error)

	CreateNode(node *corev1.Node) error
	GetNode(name string) (*corev1.Node, error)
	UpdateNode(node *corev1.Node) error
	DeleteNode(name string) error
	ListNodes() ([]*corev1.Node, error)
}

package store

import (
	"context"
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
	CreatePod(ctx context.Context, pod *corev1.Pod) error
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	UpdatePod(ctx context.Context, pod *corev1.Pod) error
	DeletePod(ctx context.Context, namespace, name string) error
	ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error)

	CreateNode(ctx context.Context, node *corev1.Node) error
	GetNode(ctx context.Context, name string) (*corev1.Node, error)
	UpdateNode(ctx context.Context, node *corev1.Node) error
	DeleteNode(ctx context.Context, name string) error
	ListNodes(ctx context.Context) ([]*corev1.Node, error)
}

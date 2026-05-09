package cli

import (
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestObjectName_Pod(t *testing.T) {
	p := &corev1.Pod{Name: "my-pod"}
	name, err := objectName(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "my-pod" {
		t.Errorf("got %q want %q", name, "my-pod")
	}
}

func TestObjectName_Node(t *testing.T) {
	n := &corev1.Node{Name: "node-1"}
	name, err := objectName(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "node-1" {
		t.Errorf("got %q want %q", name, "node-1")
	}
}

func TestObjectName_UnsupportedType(t *testing.T) {
	_, err := objectName("just a string")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

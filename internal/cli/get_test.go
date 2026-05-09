package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func marshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestPrintResource_PodList(t *testing.T) {
	pods := []corev1.Pod{
		{Name: "p1", Namespace: "default", Image: "nginx", NodeName: "node-1", Status: corev1.PodRunning},
		{Name: "p2", Namespace: "default", Image: "redis", NodeName: "node-2", Status: corev1.PodPending},
	}

	var buf bytes.Buffer
	if err := printResource(&buf, "pods", true, marshalJSON(t, pods)); err != nil {
		t.Fatalf("printResource: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "p1") || !strings.Contains(out, "p2") {
		t.Errorf("expected both pod names in output:\n%s", out)
	}
	if !strings.Contains(out, "NAMESPACE") {
		t.Errorf("expected header in output:\n%s", out)
	}
}

func TestPrintResource_PodSingle(t *testing.T) {
	pod := corev1.Pod{Name: "p1", Namespace: "kube-system", Image: "nginx", Status: corev1.PodRunning}

	var buf bytes.Buffer
	if err := printResource(&buf, "pod", false, marshalJSON(t, pod)); err != nil {
		t.Fatalf("printResource: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "p1") {
		t.Errorf("expected pod name in output:\n%s", out)
	}
	if !strings.Contains(out, "kube-system") {
		t.Errorf("expected namespace in output:\n%s", out)
	}
}

func TestPrintResource_NodeList(t *testing.T) {
	nodes := []corev1.Node{
		{Name: "node-1", Address: "10.0.0.1", Status: corev1.NodeReady},
		{Name: "node-2", Address: "10.0.0.2", Status: corev1.NodeNotReady},
	}

	var buf bytes.Buffer
	if err := printResource(&buf, "nodes", true, marshalJSON(t, nodes)); err != nil {
		t.Fatalf("printResource: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "node-1") || !strings.Contains(out, "node-2") {
		t.Errorf("expected both node names in output:\n%s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header in output:\n%s", out)
	}
}

func TestPrintResource_UnknownKind(t *testing.T) {
	var buf bytes.Buffer
	err := printResource(&buf, "deployments", true, []byte(`[]`))
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestPrintResource_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printResource(&buf, "pods", true, []byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

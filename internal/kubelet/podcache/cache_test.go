package podcache

import (
	"testing"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func TestCacheSetGetListDelete(t *testing.T) {
	cache := NewCache()
	pod := &corev1.Pod{
		Name:      "p1",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	cache.Set(pod)
	pod.Labels["app"] = "changed"

	got, ok := cache.Get("default", "p1")
	if !ok {
		t.Fatal("expected pod in cache")
	}
	if got.Labels["app"] != "web" {
		t.Errorf("label: got %q want web", got.Labels["app"])
	}

	got.Labels["app"] = "mutated"
	gotAgain, _ := cache.Get("default", "p1")
	if gotAgain.Labels["app"] != "web" {
		t.Errorf("Get should return a copy, got %q", gotAgain.Labels["app"])
	}

	if pods := cache.List(); len(pods) != 1 || pods[0].Name != "p1" {
		t.Fatalf("List: got %+v", pods)
	}

	cache.Delete("default", "p1")
	if _, ok := cache.Get("default", "p1"); ok {
		t.Fatal("expected pod to be deleted")
	}
}

func TestCacheClonesDeletionTimestamp(t *testing.T) {
	cache := NewCache()
	ts := time.Now()
	pod := &corev1.Pod{Name: "p1", Namespace: "default", DeletionTimestamp: &ts}

	cache.Set(pod)
	got, ok := cache.Get("default", "p1")
	if !ok {
		t.Fatal("expected pod in cache")
	}
	if got.DeletionTimestamp == pod.DeletionTimestamp {
		t.Fatal("expected deletion timestamp pointer to be cloned")
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return f
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeConfig(t, `
etcd:
  endpoints:
    - localhost:2379
    - localhost:2380
  dial_timeout: 5s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Etcd.Endpoints) != 2 {
		t.Errorf("endpoints: got %d want 2", len(cfg.Etcd.Endpoints))
	}
	if cfg.Etcd.Endpoints[0] != "localhost:2379" {
		t.Errorf("endpoints[0]: got %q want %q", cfg.Etcd.Endpoints[0], "localhost:2379")
	}
	if cfg.Etcd.DialTimeout != "5s" {
		t.Errorf("dial_timeout: got %q want %q", cfg.Etcd.DialTimeout, "5s")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "key: [unclosed bracket")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EmptyEndpoints(t *testing.T) {
	path := writeConfig(t, `
etcd:
  endpoints: []
  dial_timeout: 3s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Etcd.Endpoints) != 0 {
		t.Errorf("expected empty endpoints, got %v", cfg.Etcd.Endpoints)
	}
}

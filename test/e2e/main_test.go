//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/joshL1215/k8s-like/internal/apiserver"
	"github.com/joshL1215/k8s-like/internal/store/distribkv"
)

var baseURL string

func TestMain(m *testing.M) {
	ctx := context.Background()

	etcdC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "gcr.io/etcd-development/etcd:v3.5.17",
			ExposedPorts: []string{"2379/tcp"},
			Cmd: []string{
				"etcd",
				"--listen-client-urls", "http://0.0.0.0:2379",
				"--advertise-client-urls", "http://0.0.0.0:2379",
			},
			WaitingFor: wait.ForListeningPort("2379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start etcd container: %v\n", err)
		os.Exit(1)
	}
	defer etcdC.Terminate(ctx)

	host, _ := etcdC.Host(ctx)
	port, _ := etcdC.MappedPort(ctx, "2379")
	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	store := distribkv.CreateDistributedKVStore([]string{endpoint}, 10*time.Second)
	defer store.Close()

	srv := httptest.NewServer(apiserver.CreateAPIServer(store).Handler())
	defer srv.Close()
	baseURL = srv.URL

	os.Exit(m.Run())
}

// testNS returns a namespace unique to this test, safe for use in etcd keys.
func testNS(t *testing.T) string {
	t.Helper()
	return strings.ToLower(strings.NewReplacer("/", "-", "_", "-").Replace(t.Name()))
}

// podURL builds a pod endpoint URL.
func podURL(ns, name string) string {
	if name == "" {
		return fmt.Sprintf("%s/api/v1/namespace/%s/pods", baseURL, ns)
	}
	return fmt.Sprintf("%s/api/v1/namespace/%s/pods/%s", baseURL, ns, name)
}

// nodeURL builds a node endpoint URL.
func nodeURL(name string) string {
	if name == "" {
		return fmt.Sprintf("%s/api/v1/nodes", baseURL)
	}
	return fmt.Sprintf("%s/api/v1/nodes/%s", baseURL, name)
}

// do fires an HTTP request and returns the response. Caller closes body.
func do(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// decode JSON-decodes the response body into T and closes it.
func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return v
}

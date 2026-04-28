package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

const requestTimeout = 10 * time.Second

func resourcePath(kind string) (string, bool, error) {
	switch strings.ToLower(kind) {
	case "pod", "pods", "po":
		return "pods", true, nil
	case "node", "nodes", "no":
		return "nodes", false, nil
	}
	return "", false, fmt.Errorf("unknown resource %q", kind)
}

func resourceURL(kind, name string) (string, error) {
	path, namespaced, err := resourcePath(kind)
	if err != nil {
		return "", err
	}
	var url string
	if namespaced {
		url = fmt.Sprintf("%s/api/v1/namespace/%s/%s", server, namespace, path)
	} else {
		url = fmt.Sprintf("%s/api/v1/%s", server, path)
	}
	if name != "" {
		url += "/" + name
	}
	return url, nil
}

func doRequest(method, url string, body any) ([]byte, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e struct{ Error, Detail string }
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			return nil, fmt.Errorf("%s: %s", e.Error, e.Detail)
		}
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}
	return data, nil
}

func decodeManifest(data []byte) (kind string, obj any, err error) {
	var meta struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", nil, err
	}
	switch strings.ToLower(meta.Kind) {
	case "pod":
		var p corev1.Pod
		if err := json.Unmarshal(data, &p); err != nil {
			return "", nil, err
		}
		return "pods", &p, nil
	case "node":
		var n corev1.Node
		if err := json.Unmarshal(data, &n); err != nil {
			return "", nil, err
		}
		return "nodes", &n, nil
	}
	return "", nil, fmt.Errorf("unknown or missing kind in manifest")
}

package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/namespace/%s/pods/%s", namespace, name))
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Pod](resp)
}

func (c *Client) CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	resp, err := c.sendJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/namespace/%s/pods", pod.Namespace), pod)
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Pod](resp)
}

func (c *Client) UpdatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	resp, err := c.sendJSON(ctx, http.MethodPut, fmt.Sprintf("/api/v1/namespace/%s/pods/%s", pod.Namespace, pod.Name), pod)
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Pod](resp)
}

func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	resp, err := c.delete(ctx, fmt.Sprintf("/api/v1/namespace/%s/pods/%s", namespace, name))
	if err != nil {
		return err
	}
	drain(resp)
	return nil
}

func (c *Client) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/namespace/%s/pods", namespace))
	if err != nil {
		return nil, err
	}
	return decode[[]*corev1.Pod](resp)
}

func (c *Client) WatchPods(ctx context.Context, namespace, nodeName string) (<-chan corev1.WatchEvent, error) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/v1/namespace/%s/pods", c.baseURL, namespace))
	q := u.Query()
	q.Set("watch", "true")
	if nodeName != "" {
		q.Set("nodeName", nodeName)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("watch pods: server returned %s", resp.Status)
	}

	ch := make(chan corev1.WatchEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for {
			var ev corev1.WatchEvent
			if err := dec.Decode(&ev); err != nil {
				return
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

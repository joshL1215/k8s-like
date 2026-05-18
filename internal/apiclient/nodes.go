package apiclient

import (
	"context"
	"fmt"
	"net/http"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
)

func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/nodes/%s", name))
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Node](resp)
}

func (c *Client) CreateNode(ctx context.Context, node *corev1.Node) (*corev1.Node, error) {
	resp, err := c.sendJSON(ctx, http.MethodPost, "/api/v1/nodes", node)
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Node](resp)
}

func (c *Client) UpdateNode(ctx context.Context, node *corev1.Node) (*corev1.Node, error) {
	resp, err := c.sendJSON(ctx, http.MethodPut, fmt.Sprintf("/api/v1/nodes/%s", node.Name), node)
	if err != nil {
		return nil, err
	}
	return decode[*corev1.Node](resp)
}

func (c *Client) DeleteNode(ctx context.Context, name string) error {
	resp, err := c.delete(ctx, fmt.Sprintf("/api/v1/nodes/%s", name))
	if err != nil {
		return err
	}
	drain(resp)
	return nil
}

func (c *Client) ListNodes(ctx context.Context) ([]*corev1.Node, error) {
	resp, err := c.get(ctx, "/api/v1/nodes")
	if err != nil {
		return nil, err
	}
	return decode[[]*corev1.Node](resp)
}

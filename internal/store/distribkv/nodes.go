package distribkv

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/store"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const nodePrefix = "/registry/nodes"

func buildNodeKey(name string) string {
	return nodePrefix + "/" + name
}

func (s *DistributedKVStore) CreateNode(ctx context.Context, node *corev1.Node) error {
	cli, nodeKey := s.etcd, buildNodeKey(node.Name)
	nodeVal, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("node marshal: %w", err)
	}

	// Txn (etcd transaction) ensures atomicity in if-then
	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(nodeKey), "=", 0)).
		Then(clientv3.OpPut(nodeKey, string(nodeVal))).Commit()
	if err != nil {
		return fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return store.ErrNodeExists
	}
	return nil
}

func (s *DistributedKVStore) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	cli, nodeKey := s.etcd, buildNodeKey(name)

	resp, err := cli.Get(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("node get: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return nil, store.ErrNodeNotExist
	}

	node := &corev1.Node{}
	if err := json.Unmarshal(resp.Kvs[0].Value, node); err != nil {
		return nil, fmt.Errorf("unmarshal node: %w", err)
	}
	return node, nil
}

func (s *DistributedKVStore) UpdateNode(ctx context.Context, node *corev1.Node) error {
	cli, nodeKey := s.etcd, buildNodeKey(node.Name)

	nodeVal, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("node marshal: %w", err)
	}

	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(nodeKey), "!=", 0)).
		Then(clientv3.OpPut(nodeKey, string(nodeVal))).Commit()
	if err != nil {
		return fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return store.ErrNodeNotExist
	}
	return nil
}

func (s *DistributedKVStore) DeleteNode(ctx context.Context, name string) error {
	cli, nodeKey := s.etcd, buildNodeKey(name)

	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(nodeKey), "!=", 0)).
		Then(clientv3.OpDelete(nodeKey)).Commit()
	if err != nil {
		return fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return store.ErrNodeNotExist
	}
	return nil
}

func (s *DistributedKVStore) ListNodes(ctx context.Context) ([]*corev1.Node, error) {
	cli := s.etcd
	keyPrefix := nodePrefix + "/"

	resp, err := cli.Get(ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("node list: %w", err)
	}

	nodes := make([]*corev1.Node, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		node := &corev1.Node{}
		if err := json.Unmarshal(kv.Value, node); err != nil {
			return nil, fmt.Errorf("unmarshal node: %w", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

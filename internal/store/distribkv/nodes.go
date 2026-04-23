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

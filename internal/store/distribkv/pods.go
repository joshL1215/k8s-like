package distribkv

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const podPrefix = "/registry/pods/"

func buildPodKey(namespace, name string) string {
	return podPrefix + namespace + "/" + name
}

func (s *DistributedKVStore) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	cli, podKey := s.etcd, buildPodKey(pod.Namespace, pod.Name)
	podVal, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed pod marshal: %w", err)
	}

	// Txn (etcd transaction) ensures atomicity in if-then
	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(podKey), "=", 0)).
		Then(clientv3.OpPut(podKey, string(podVal))).Commit()
	if err != nil {
		return fmt.Errorf("failed etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return ErrPodExists
	}
	return nil
}

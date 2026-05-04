package distribkv

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/store"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const podPrefix = "/registry/pods/"

func buildPodKey(namespace, name string) string {
	return podPrefix + namespace + "/" + name
}

func (s *DistributedKVStore) CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	cli, podKey := s.etcd, buildPodKey(pod.Namespace, pod.Name)

	podVal, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("pod marshal: %w", err)
	}

	// Txn (etcd transaction) ensures atomicity in if-then
	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(podKey), "=", 0)).
		Then(clientv3.OpPut(podKey, string(podVal))).Commit()
	if err != nil {
		return nil, fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return nil, store.ErrPodExists
	}
	storedPod := new(corev1.Pod)
	*storedPod = *pod
	return storedPod, nil
}

func (s *DistributedKVStore) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	cli, podKey := s.etcd, buildPodKey(namespace, name)

	resp, err := cli.Get(ctx, podKey)
	if err != nil {
		return nil, fmt.Errorf("pod get: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return nil, store.ErrPodNotExist
	}

	pod := &corev1.Pod{}
	if err := json.Unmarshal(resp.Kvs[0].Value, pod); err != nil {
		return nil, fmt.Errorf("unmarshal pod: %w", err)
	}
	return pod, nil
}

func (s *DistributedKVStore) UpdatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	cli, podKey := s.etcd, buildPodKey(pod.Namespace, pod.Name)

	podVal, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("pod marshal: %w", err)
	}

	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(podKey), "!=", 0)).
		Then(clientv3.OpPut(podKey, string(podVal))).Commit()
	if err != nil {
		return nil, fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return nil, store.ErrPodNotExist
	}
	storedPod := new(corev1.Pod)
	*storedPod = *pod
	return storedPod, nil
}

func (s *DistributedKVStore) DeletePod(ctx context.Context, namespace, name string) error {
	cli, podKey := s.etcd, buildPodKey(namespace, name)

	resp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(podKey), "!=", 0)).
		Then(clientv3.OpDelete(podKey)).Commit()
	if err != nil {
		return fmt.Errorf("etcd transaction: %w", err)
	}
	if !resp.Succeeded {
		return store.ErrPodNotExist
	}
	return nil
}

func (s *DistributedKVStore) ListPods(ctx context.Context, namespace string) ([]*corev1.Pod, error) {
	cli := s.etcd
	keyPrefix := podPrefix + namespace + "/"

	resp, err := cli.Get(ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("pod list: %w", err)
	}

	pods := make([]*corev1.Pod, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		pod := &corev1.Pod{}
		if err := json.Unmarshal(kv.Value, pod); err != nil {
			return nil, fmt.Errorf("unmarshal pod: %w", err)
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

package distribkv

import (
	"context"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type DistributedKVStore struct {
	etcd *clientv3.Client
}

func CreateDistributedKVStore() *DistributedKVStore {
	endpoint := "localhost:2379"
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Status(ctx, endpoint); err != nil {
		log.Fatalf("Failed to connect to etcd at %s: %v", endpoint, err)
	}

	log.Print("Successfully connected to etcd.")
	return &DistributedKVStore{
		etcd: cli,
	}
}

func (s *DistributedKVStore) Close() error {
	return s.etcd.Close()
}

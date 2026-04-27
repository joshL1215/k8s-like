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

func CreateDistributedKVStore(endpoints []string, dialTimeout time.Duration) *DistributedKVStore {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})

	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if _, err := cli.Status(ctx, endpoints[0]); err != nil {
		log.Fatalf("Failed to connect to etcd at %s: %v", endpoints[0], err)
	}

	log.Print("Successfully connected to etcd.")
	return &DistributedKVStore{
		etcd: cli,
	}
}

func (s *DistributedKVStore) Close() error {
	return s.etcd.Close()
}

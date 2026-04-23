package distribkv

import (
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type DistributedKVStore struct {
	etcd *clientv3.Client
}

func CreateDistributedKVStore() *DistributedKVStore {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:8080"},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	log.Print("Successfully connected to etcd.")
	return &DistributedKVStore{
		etcd: cli,
	}
}

func (s *DistributedKVStore) Close() error {
	return s.etcd.Close()
}

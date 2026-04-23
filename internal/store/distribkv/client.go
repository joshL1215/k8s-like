package distribkv

import (
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type DistributedKVStore struct {
	client     *clientv3.Client
	podPrefix  string
	nodePrefix string
}

func CreateDistributedKVStore() *DistributedKVStore {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:8080"},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	defer cli.Close()
	log.Print("Successfully connected to etcd.")
	return &DistributedKVStore{}
}

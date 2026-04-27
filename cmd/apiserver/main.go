package main

import (
	"flag"
	"log"
	"time"

	"github.com/joshL1215/k8s-like/internal/apiserver"
	"github.com/joshL1215/k8s-like/internal/config"
	"github.com/joshL1215/k8s-like/internal/store/distribkv"
)

func main() {
	configPath := flag.String("config", "cmd/apiserver/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dialTimeout, err := time.ParseDuration(cfg.Etcd.DialTimeout)
	if err != nil {
		log.Fatalf("parse etcd.dial_timeout: %v", err)
	}

	s := distribkv.CreateDistributedKVStore(cfg.Etcd.Endpoints, dialTimeout)
	defer s.Close()
	apiserver.CreateAPIServer(s).Serve(":5173")
}

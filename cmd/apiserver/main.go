package apiserver

import (
	"github.com/joshL1215/k8s-like/internal/apiserver"
	"github.com/joshL1215/k8s-like/internal/store/distribkv"
)

func main() {
	s := distribkv.CreateDistributedKVStore()
	defer s.Close()
	apiserver.CreateAPIServer(s).Serve(":5173")
}

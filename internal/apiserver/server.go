package apiserver

import (
	"log"
	"net/http"
	"time"

	"github.com/joshL1215/k8s-like/internal/apiserver/nodes"
	"github.com/joshL1215/k8s-like/internal/apiserver/watchers"
	"github.com/joshL1215/k8s-like/internal/store"
)

const DefaultNamespace = "default"

type APIServer struct {
	router       *http.ServeMux
	server       *http.Server
	store        store.StoreInterface // having an interface here makes it store-implementation-agnostic
	watchManager *watchers.WatchManager
}

func CreateAPIServer(s store.StoreInterface) *APIServer {
	mux := http.NewServeMux()
	apiServer := &APIServer{
		router:       mux,
		store:        s,
		watchManager: watchers.NewWatchManager(),
	}

	apiServer.registerRoutes()
	return apiServer
}

func (s *APIServer) Serve(port string) {
	s.server = &http.Server{
		Addr:              port,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Serving API server...")
	if err := s.server.ListenAndServe(); err != nil {
		log.Printf("Could not serve API server: %v", err)
	}
}

func (s *APIServer) registerRoutes() {
	nodeHandler := nodes.NewHandler(s.store)
	nodeHandler.Register(s.router)
}

package nodes

import "net/http"

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/nodes", h.create)
	mux.HandleFunc("GET /api/v1/nodes", h.list)
	mux.HandleFunc("GET /api/v1/nodes/{name}", h.get)
	mux.HandleFunc("PUT /api/v1/nodes/{name}", h.update)
	mux.HandleFunc("DELETE /api/v1/nodes/{name}", h.delete)
}

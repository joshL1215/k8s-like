package pods

import "net/http"

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/namespace/{namespace}/pods", h.create)
	mux.HandleFunc("GET /api/v1/namespace/{namespace}/pods", h.list)
	mux.HandleFunc("GET /api/v1/namespace/{namespace}/pods/{podname}", h.get)
	mux.HandleFunc("PUT /api/v1/namespace/{namespace}/pods/{podname}", h.update)
	mux.HandleFunc("DELETE /api/v1/pods/{name}", h.delete)
}

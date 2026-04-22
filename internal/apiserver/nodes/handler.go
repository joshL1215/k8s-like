package nodes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/store"
)

type Handler struct {
	store store.StoreInterface
}

func NewHandler(s store.StoreInterface) *Handler {
	return &Handler{store: s}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	writeJSON(w, code, map[string]string{"error": msg, "detail": detail})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var node corev1.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if node.Name == "" {
		writeErr(w, http.StatusBadRequest, "A node name must be provided", nil)
		return
	}
	if node.Status == "" {
		node.Status = corev1.NodeReady
	}

	if err := h.store.CreateNode(&node); err != nil {
		log.Printf("Error creating node %s: %v", node.Name, err)
		if errors.Is(err, store.ErrNodeExists) {
			writeErr(w, http.StatusConflict, "Failed to create node", err)
		} else {
			writeErr(w, http.StatusInternalServerError, "Failed to create node", err)
		}
		return
	}
	log.Printf("Created node %s successfully", node.Name)
	writeJSON(w, http.StatusCreated, node)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	node, err := h.store.GetNode(name)
	if err != nil {
		if errors.Is(err, store.ErrNodeNotExist) {
			writeErr(w, http.StatusNotFound, "Node not found", err)
		} else {
			writeErr(w, http.StatusInternalServerError, "Failed to get node", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var node corev1.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if node.Name == "" {
		writeErr(w, http.StatusBadRequest, "A node name must be provided", nil)
		return
	}

	if _, err := h.store.GetNode(node.Name); err != nil {
		if errors.Is(err, store.ErrNodeNotExist) {
			writeErr(w, http.StatusNotFound, "Node does not exist", err)
		} else {
			writeErr(w, http.StatusInternalServerError, "Failed to find node", err)
		}
		return
	}

	if err := h.store.UpdateNode(&node); err != nil {
		log.Printf("Failed to update node: %v", err)
		writeErr(w, http.StatusInternalServerError, "Failed to update node", err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.store.DeleteNode(name); err != nil {
		log.Printf("Error deleting node %s: %v", name, err)
		if errors.Is(err, store.ErrNodeNotExist) {
			writeErr(w, http.StatusNotFound, "Node not found for deletion", err)
		} else {
			writeErr(w, http.StatusInternalServerError, "Unable to delete node", err)
		}
		return
	}
	log.Printf("Node %s successfully deleted", name)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Node " + name + " successfully deleted"})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.store.ListNodes()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Unable to list nodes", err)
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

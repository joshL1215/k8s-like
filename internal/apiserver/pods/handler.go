package pods

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	corev1 "github.com/joshL1215/k8s-like/api/core/v1"
	"github.com/joshL1215/k8s-like/internal/apiserver/httpx"
	"github.com/joshL1215/k8s-like/internal/store"
)

type Handler struct {
	store store.StoreInterface
}

func NewHandler(s store.StoreInterface) *Handler {
	return &Handler{store: s}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var pod corev1.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		httpx.WriteErr(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if pod.Name == "" {
		httpx.WriteErr(w, http.StatusBadRequest, "A pod name must be provided", nil)
		return
	}
	if pod.Namespace == "" {
		pod.Namespace = "default"
	}
	if pod.Status == "" {
		pod.Status = corev1.PodPending
	}
	if err := h.store.CreatePod(ctx, &pod); err != nil {
		log.Printf("Error creating pod %s: %v", pod.Name, err)
		if errors.Is(err, store.ErrPodExists) {
			httpx.WriteErr(w, http.StatusConflict, "Failed to create pod", err)
		} else {
			httpx.WriteErr(w, http.StatusInternalServerError, "Failed to create pod", err)
		}
		return
	}
	log.Printf("Created pod %s successfully", pod.Name)
	httpx.WriteJSON(w, http.StatusCreated, pod)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	pod, err := h.store.GetPod(ctx, namespace, name)
	if err != nil {
		if errors.Is(err, store.ErrPodNotExist) {
			httpx.WriteErr(w, http.StatusNotFound, "Pod not found", err)
		} else {
			httpx.WriteErr(w, http.StatusInternalServerError, "Failed to get pod", err)
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pod)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var pod corev1.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		httpx.WriteErr(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if pod.Namespace == "" {
		httpx.WriteErr(w, http.StatusBadRequest, "A namespace must be provided", nil)
		return
	}
	if pod.Name == "" {
		httpx.WriteErr(w, http.StatusBadRequest, "A pod name must be provided", nil)
		return
	}
	if _, err := h.store.GetPod(ctx, pod.Namespace, pod.Name); err != nil {
		if errors.Is(err, store.ErrPodNotExist) {
			httpx.WriteErr(w, http.StatusNotFound, "Pod does not exist", err)
		} else {
			httpx.WriteErr(w, http.StatusInternalServerError, "Failed to find pod", err)
		}
		return
	}
	if err := h.store.UpdatePod(ctx, &pod); err != nil {
		log.Printf("Failed to update pod: %v", err)
		httpx.WriteErr(w, http.StatusInternalServerError, "Failed to update pod", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pod)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	if err := h.store.DeletePod(ctx, namespace, name); err != nil {
		log.Printf("Error deleting pod %s: %v", name, err)
		if errors.Is(err, store.ErrPodNotExist) {
			httpx.WriteErr(w, http.StatusNotFound, "Pod not found for deletion", err)
		} else {
			httpx.WriteErr(w, http.StatusInternalServerError, "Unable to delete pod", err)
		}
		return
	}
	log.Printf("Pod %s successfully deleted", name)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "Pod " + name + "in namespace " + namespace + " successfully deleted"})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := r.PathValue("namespace")
	nodeName := r.URL.Query().Get("nodeName")
	pods, err := h.store.ListPods(ctx, namespace)
	if err != nil {
		httpx.WriteErr(w, http.StatusInternalServerError, "Unable to list pods in namespace "+namespace, err)
		return
	}

	if nodeName != "" {
		var filteredPods []*corev1.Pod
		for _, pod := range pods {
			if pod.NodeName == nodeName {
				filteredPods = append(filteredPods, pod)
			}
		}
		httpx.WriteJSON(w, http.StatusOK, filteredPods)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, pods)
}

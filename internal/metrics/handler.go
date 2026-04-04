package metrics

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	registry *Registry
}

func NewHandler(registry *Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := h.registry.Render()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

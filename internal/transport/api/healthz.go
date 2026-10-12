package api

import (
	"net/http"
)

func (h *ChatHandler) HealthZ(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

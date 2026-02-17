package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/srgjo27/scalable_ticket/internal/core/services"
)

type BookingHandler struct {
	svc *services.BookingService
}

func NewBookingHandler(svc *services.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed}`, http.StatusMethodNotAllowed)
		return
	}

	var req services.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid json body"})
		return
	}

	resp, err := h.svc.CreateBooking(r.Context(), req)

	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "seat not found") || strings.Contains(errMsg, "not available") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		} else if strings.Contains(errMsg, "invalid") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {

	}
}

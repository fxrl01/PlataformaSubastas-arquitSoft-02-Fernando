package handlers

import (
	"errors"
	"net/http"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type BidHistoryHandler struct {
	service *services.BidHistoryService
}

func NewBidHistoryHandler(service *services.BidHistoryService) *BidHistoryHandler {
	return &BidHistoryHandler{service: service}
}

func (h *BidHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	bids, err := h.service.List(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrSubastaNoEncontrada) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, bids)
}

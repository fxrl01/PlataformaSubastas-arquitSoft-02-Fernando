package handlers

import (
	"errors"
	"net/http"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type BidHandler struct {
	bids *services.AuctionBidService
}

func NewBidHandler(bids *services.AuctionBidService) *BidHandler {
	return &BidHandler{bids: bids}
}

type placeBidRequest struct {
	Monto float64 `json:"monto"`
}

func (h *BidHandler) Place(w http.ResponseWriter, r *http.Request) {
	var request placeBidRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	response, err := h.bids.PlaceBid(r.Context(), services.PlaceBidCmd{
		SubastaID: r.PathValue("id"), PostorID: user.ID, Monto: request.Monto,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrPujaInferior), errors.Is(err, domain.ErrSubastaCerrada),
			errors.Is(err, domain.ErrPostorIgual):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "bid rejected"})
		case errors.Is(err, domain.ErrPujaConcurrente):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "auction changed, retry bid"})
		case errors.Is(err, domain.ErrLockTimeout):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "auction busy, retry bid"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

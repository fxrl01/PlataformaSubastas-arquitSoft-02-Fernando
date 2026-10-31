package handlers

import (
	"errors"
	"net/http"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type GuaranteeHandler struct {
	service *services.GuaranteeService
}

func NewGuaranteeHandler(service *services.GuaranteeService) *GuaranteeHandler {
	return &GuaranteeHandler{service: service}
}

type createGuaranteeRequest struct {
	Amount float64 `json:"monto"`
}

func (h *GuaranteeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createGuaranteeRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	view, err := h.service.Create(r.Context(), services.CreateGuaranteeCmd{
		AuctionID: r.PathValue("id"), BidderID: user.ID, Amount: request.Amount,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrGuaranteeInsufficient):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "guarantee insufficient"})
		case errors.Is(err, domain.ErrGuaranteeAlreadyActive):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "guarantee already active"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

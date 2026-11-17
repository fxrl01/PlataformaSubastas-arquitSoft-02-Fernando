package handlers

import (
	"errors"
	"io"
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

type prepareStripeIntentRequest struct {
	Amount float64 `json:"monto,omitempty"`
}

type createGuaranteeRequest struct {
	Amount          float64 `json:"monto"`
	WalletAddress   string  `json:"wallet_address,omitempty"`
	PaymentIntentID string  `json:"payment_intent_id,omitempty"`
}

func (h *GuaranteeHandler) PrepareStripeIntent(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	var request prepareStripeIntentRequest
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &request) {
			return
		}
	}
	offer, err := h.service.PrepareStripeIntent(r.Context(), r.PathValue("id"), user.ID, request.Amount)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrGuaranteeAlreadyActive):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "guarantee already active"})
		case errors.Is(err, domain.ErrStripeHoldFailed):
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusOK, offer)
}

func (h *GuaranteeHandler) GetMyGuarantee(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	view, err := h.service.GetActive(r.Context(), r.PathValue("id"), user.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no active guarantee found"})
		return
	}
	writeJSON(w, http.StatusOK, view)
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
	wallet := request.WalletAddress
	if wallet == "" {
		wallet = r.Header.Get("X-Web3-Wallet")
	}
	view, err := h.service.Create(r.Context(), services.CreateGuaranteeCmd{
		AuctionID:       r.PathValue("id"),
		BidderID:        user.ID,
		Amount:          request.Amount,
		WalletAddress:   wallet,
		PaymentIntentID: request.PaymentIntentID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrInvalidWeb3Wallet):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid web3 wallet address"})
		case errors.Is(err, domain.ErrGuaranteeInsufficient):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "guarantee insufficient"})
		case errors.Is(err, domain.ErrGuaranteeAlreadyActive):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "guarantee already active"})
		case errors.Is(err, domain.ErrStripeHoldFailed):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (h *GuaranteeHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid webhook payload"})
		return
	}
	sigHeader := r.Header.Get("Stripe-Signature")
	eventType, err := h.service.HandleStripeWebhook(r.Context(), payload, sigHeader)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid stripe webhook signature"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"received":   true,
		"event_type": eventType,
	})
}

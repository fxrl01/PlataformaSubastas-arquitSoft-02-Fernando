package handlers

import (
	"errors"
	"net/http"
	"strconv"

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
	Monto          float64 `json:"monto"`
	WalletAddress  string  `json:"wallet_address,omitempty"`
	Signature      string  `json:"signature,omitempty"`
	Nonce          string  `json:"nonce,omitempty"`
	Deadline       int64   `json:"deadline,omitempty"`
	AgentMandateID string  `json:"mandato_agente_id,omitempty"`
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
	wallet := request.WalletAddress
	if wallet == "" {
		wallet = r.Header.Get("X-Web3-Wallet")
	}
	signature := request.Signature
	if signature == "" {
		signature = r.Header.Get("X-Web3-Signature")
	}
	nonce := request.Nonce
	if nonce == "" {
		nonce = r.Header.Get("X-Web3-Nonce")
	}
	deadline := request.Deadline
	if deadline == 0 {
		if rawDeadline := r.Header.Get("X-Web3-Deadline"); rawDeadline != "" {
			deadline, _ = strconv.ParseInt(rawDeadline, 10, 64)
		}
	}
	mandateID := request.AgentMandateID
	if mandateID == "" {
		mandateID = r.Header.Get("X-AI-Agent-Mandate")
	}
	response, err := h.bids.PlaceBid(r.Context(), services.PlaceBidCmd{
		SubastaID:      r.PathValue("id"),
		PostorID:       user.ID,
		Monto:          request.Monto,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		WalletAddress:  wallet,
		Signature:      signature,
		Nonce:          nonce,
		Deadline:       deadline,
		AgentMandateID: mandateID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrInvalidWeb3Wallet), errors.Is(err, domain.ErrInvalidWeb3Signature):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid web3 eip712 signature or wallet"})
		case errors.Is(err, domain.ErrAgentMandateInvalid), errors.Is(err, domain.ErrAgentMandateExpired),
			errors.Is(err, domain.ErrAgentMandateExceeded):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "ai agent mandate exceeded or invalid"})
		case errors.Is(err, domain.ErrPujaInferior), errors.Is(err, domain.ErrSubastaCerrada),
			errors.Is(err, domain.ErrPostorIgual), errors.Is(err, domain.ErrGarantiaInsuficiente):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "bid rejected"})
		case errors.Is(err, domain.ErrPujaConcurrente):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "auction changed, retry bid"})
		case errors.Is(err, domain.ErrIdempotencyKeyConflict):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency key reused with another amount"})
		case errors.Is(err, domain.ErrLockTimeout):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "auction busy, retry bid"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

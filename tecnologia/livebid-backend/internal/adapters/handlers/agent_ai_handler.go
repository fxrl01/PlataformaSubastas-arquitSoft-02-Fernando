package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type AgentAIHandler struct {
	delegation *services.AgentDelegationService
	router     *ai.MultiModelRouter
	breaker    *ai.CircuitBreakerIA
	detector   *ai.AsyncFraudDetector
	assistant  *ai.GeminiAssistant
	auctions   domain.SubastaRepository
}

func NewAgentAIHandler(
	delegation *services.AgentDelegationService,
	router *ai.MultiModelRouter,
	breaker *ai.CircuitBreakerIA,
	detector *ai.AsyncFraudDetector,
) *AgentAIHandler {
	return &AgentAIHandler{
		delegation: delegation,
		router:     router,
		breaker:    breaker,
		detector:   detector,
	}
}

func (h *AgentAIHandler) WithAssistant(assistant *ai.GeminiAssistant, auctions domain.SubastaRepository) *AgentAIHandler {
	h.assistant = assistant
	h.auctions = auctions
	return h
}

type createMandateRequest struct {
	AgentID             string               `json:"agente_id"`
	MaxAuthorizedAmount float64              `json:"monto_maximo_autorizado"`
	Strategy            domain.AgentStrategy `json:"estrategia,omitempty"`
}

type autoBidRequest struct {
	MandateID string `json:"mandato_id"`
}

type switchModelRequest struct {
	Model string `json:"modelo"`
}

func (h *AgentAIHandler) Delegate(w http.ResponseWriter, r *http.Request) {
	var req createMandateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	mandate, err := h.delegation.CreateMandate(r.Context(), services.CreateMandateCmd{
		AuctionID:           r.PathValue("id"),
		PrincipalBidderID:   user.ID,
		AgentID:             req.AgentID,
		MaxAuthorizedAmount: req.MaxAuthorizedAmount,
		Strategy:            req.Strategy,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrSubastaCerrada):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "auction not accepting bids"})
		case errors.Is(err, domain.ErrAgentMandateInvalid):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent mandate parameters"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, mandate)
}

func (h *AgentAIHandler) AutoBid(w http.ResponseWriter, r *http.Request) {
	var req autoBidRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	mandateID := req.MandateID
	if mandateID == "" {
		mandateID = r.Header.Get("X-AI-Agent-Mandate")
	}
	res, err := h.delegation.ExecuteAutonomousBid(r.Context(), services.ExecuteAutoBidCmd{
		AuctionID: r.PathValue("id"),
		BidderID:  user.ID,
		MandateID: mandateID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubastaNoEncontrada):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		case errors.Is(err, domain.ErrAgentMandateInvalid), errors.Is(err, domain.ErrAgentMandateExpired),
			errors.Is(err, domain.ErrAgentMandateExceeded):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "ai agent mandate exceeded or invalid"})
		default:
			writeJSON(w, http.StatusConflict, map[string]string{"error": "autonomous bid rejected"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *AgentAIHandler) IntelligenceReport(w http.ResponseWriter, r *http.Request) {
	auctionID := r.PathValue("id")
	var alerts []domain.FraudAlert
	if h.detector != nil {
		alerts = h.detector.ListAlertsByAuction(auctionID)
	}
	activeModel := ""
	var models []string
	if h.router != nil {
		activeModel = h.router.ActiveModel()
		models = h.router.AvailableModels()
	}
	cbState := ""
	if h.breaker != nil {
		cbState = string(h.breaker.State())
	}
	activeAgents := 0
	if h.delegation != nil {
		activeAgents = h.delegation.CountActiveMandates(auctionID, time.Now())
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subasta_id":                auctionID,
		"modelo_ia_activo":          activeModel,
		"modelos_ia_disponibles":    models,
		"circuit_breaker_ia":        cbState,
		"agentes_delegados_activos": activeAgents,
		"alertas_fraude":            alerts,
	})
}

func (h *AgentAIHandler) SwitchModel(w http.ResponseWriter, r *http.Request) {
	var req switchModelRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if h.router == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "ai router not configured"})
		return
	}
	if err := h.router.SetActiveModel(req.Model); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown ai model"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"modelo_ia_activo": h.router.ActiveModel(),
		"status":           "switched",
	})
}

type assistantChatRequest struct {
	Message      string             `json:"mensaje"`
	AuctionID    string             `json:"subasta_id,omitempty"`
	UserRole     string             `json:"rol_usuario,omitempty"`
	UserName     string             `json:"nombre_usuario,omitempty"`
	HasGuarantee bool               `json:"tiene_garantia,omitempty"`
	History      []ai.AssistantTurn `json:"historial,omitempty"`
}

func (h *AgentAIHandler) AssistantChat(w http.ResponseWriter, r *http.Request) {
	var req assistantChatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	assistant := h.assistant
	if assistant == nil {
		assistant = ai.NewGeminiAssistant("")
	}

	var aucCtx ai.AssistantAuctionContext
	if h.auctions != nil {
		if list, err := h.auctions.Listar(r.Context()); err == nil {
			aucCtx.TotalCatalog = len(list)
		}
		if req.AuctionID != "" {
			if sub, err := h.auctions.ObtenerPorID(r.Context(), req.AuctionID); err == nil && sub != nil {
				aucCtx.ID = sub.ID
				aucCtx.Title = sub.Titulo
				aucCtx.Category = sub.DetallesLote.Categoria
				aucCtx.Status = string(sub.Estado)
				aucCtx.BasePrice = sub.PrecioBase
				aucCtx.CurrentPrice = sub.PrecioActual
				aucCtx.NextMinimumBid = sub.PrecioActual + sub.CalcularIncrementoMinimo()
				aucCtx.RequiredDeposit = domain.RequiredGuarantee(sub.PrecioBase)
			}
		}
	}

	reply, err := assistant.Answer(r.Context(), ai.AssistantQuery{
		Message:        req.Message,
		UserRole:       req.UserRole,
		UserName:       req.UserName,
		HasGuarantee:   req.HasGuarantee,
		AuctionContext: aucCtx,
		History:        req.History,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mensaje requerido para el asistente"})
		return
	}
	writeJSON(w, http.StatusOK, reply)
}


package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type AuctionHandler struct {
	catalog   *services.AuctionCatalogService
	web3Audit *services.Web3AuditService
}

func NewAuctionHandler(catalog *services.AuctionCatalogService, audit ...*services.Web3AuditService) *AuctionHandler {
	var auditService *services.Web3AuditService
	if len(audit) > 0 {
		auditService = audit[0]
	}
	return &AuctionHandler{catalog: catalog, web3Audit: auditService}
}

type createAuctionRequest struct {
	Titulo       string                   `json:"titulo"`
	Descripcion  string                   `json:"descripcion"`
	MartilleroID string                   `json:"martillero_id,omitempty"`
	PrecioBase   float64                  `json:"precio_base"`
	Inicio       time.Time                `json:"inicio"`
	Cierre       time.Time                `json:"cierre"`
	Encargado    *domain.EncargadoSubasta `json:"encargado,omitempty"`
	DetallesLote *domain.DetallesLote     `json:"detalles_lote,omitempty"`
}

type auctionView struct {
	ID            string                  `json:"id"`
	VendedorID    string                  `json:"vendedor_id,omitempty"`
	MartilleroID  string                  `json:"martillero_id,omitempty"`
	Titulo        string                  `json:"titulo"`
	Descripcion   string                  `json:"descripcion"`
	PrecioBase    float64                 `json:"precio_base"`
	PrecioActual  float64                 `json:"precio_actual"`
	PostorLiderID string                  `json:"postor_lider_id,omitempty"`
	Estado        domain.EstadoSubasta    `json:"estado"`
	Inicio        time.Time               `json:"inicio"`
	Cierre        time.Time               `json:"cierre"`
	Encargado     domain.EncargadoSubasta `json:"encargado,omitempty"`
	DetallesLote  domain.DetallesLote     `json:"detalles_lote,omitempty"`
}

func (h *AuctionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createAuctionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	subasta, err := h.catalog.Crear(r.Context(), services.CrearSubastaCmd{
		ID:           "sub_" + hex.EncodeToString(idBytes),
		VendedorID:   user.ID,
		MartilleroID: request.MartilleroID,
		Titulo:       request.Titulo,
		Descripcion:  request.Descripcion,
		PrecioBase:   request.PrecioBase,
		Inicio:       request.Inicio,
		Cierre:       request.Cierre,
		Encargado:    request.Encargado,
		DetallesLote: request.DetallesLote,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSubastaInvalida) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid auction data"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusCreated, toAuctionView(subasta))
}

func (h *AuctionHandler) UpdateEncargado(w http.ResponseWriter, r *http.Request) {
	var request domain.EncargadoSubasta
	if !decodeJSON(w, r, &request) {
		return
	}
	subasta, err := h.catalog.ActualizarEncargado(r.Context(), r.PathValue("id"), request)
	if err != nil {
		if errors.Is(err, domain.ErrSubastaNoEncontrada) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
			return
		}
		if errors.Is(err, domain.ErrSubastaInvalida) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid encargado data"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, toAuctionView(subasta))
}

func (h *AuctionHandler) UpdateDetallesLote(w http.ResponseWriter, r *http.Request) {
	var request domain.DetallesLote
	if !decodeJSON(w, r, &request) {
		return
	}
	subasta, err := h.catalog.ActualizarDetallesLote(r.Context(), r.PathValue("id"), request)
	if err != nil {
		if errors.Is(err, domain.ErrSubastaNoEncontrada) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
			return
		}
		if errors.Is(err, domain.ErrSubastaInvalida) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lot details data"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, toAuctionView(subasta))
}

func (h *AuctionHandler) List(w http.ResponseWriter, r *http.Request) {
	subastas, err := h.catalog.Listar(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	views := make([]auctionView, 0, len(subastas))
	for _, subasta := range subastas {
		views = append(views, toAuctionView(subasta))
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *AuctionHandler) Get(w http.ResponseWriter, r *http.Request) {
	subasta, err := h.catalog.Obtener(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrSubastaNoEncontrada) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, toAuctionView(subasta))
}

func (h *AuctionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.catalog.Activar)
}

func (h *AuctionHandler) StartCountdown(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.catalog.IniciarConteo)
}

func (h *AuctionHandler) Close(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.catalog.Cerrar)
}

func (h *AuctionHandler) AuditWeb3(w http.ResponseWriter, r *http.Request) {
	if h.web3Audit == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "web3 audit not configured"})
		return
	}
	var proofIdx *int
	if rawIdx := r.URL.Query().Get("indice"); rawIdx != "" {
		parsed, err := strconv.Atoi(rawIdx)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid merkle leaf index"})
			return
		}
		proofIdx = &parsed
	}
	report, err := h.web3Audit.InspectAuction(r.Context(), r.PathValue("id"), proofIdx)
	if errors.Is(err, domain.ErrSubastaNoEncontrada) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *AuctionHandler) transition(w http.ResponseWriter, r *http.Request, action func(context.Context, string) error) {
	if err := action(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrSubastaNoEncontrada) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid auction state"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toAuctionView(subasta *domain.Subasta) auctionView {
	return auctionView{
		ID:            subasta.ID,
		VendedorID:    subasta.VendedorID,
		MartilleroID:  subasta.MartilleroID,
		Titulo:        subasta.Titulo,
		Descripcion:   subasta.Descripcion,
		PrecioBase:    subasta.PrecioBase,
		PrecioActual:  subasta.PrecioActual,
		PostorLiderID: subasta.PostorLiderID,
		Estado:        subasta.Estado,
		Inicio:        subasta.FechaInicio,
		Cierre:        subasta.FechaCierre,
		Encargado:     subasta.Encargado,
		DetallesLote:  subasta.DetallesLote,
	}
}

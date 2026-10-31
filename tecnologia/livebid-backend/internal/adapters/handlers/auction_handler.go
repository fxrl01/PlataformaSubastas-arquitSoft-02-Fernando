package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type AuctionHandler struct {
	catalog *services.AuctionCatalogService
}

func NewAuctionHandler(catalog *services.AuctionCatalogService) *AuctionHandler {
	return &AuctionHandler{catalog: catalog}
}

type createAuctionRequest struct {
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	PrecioBase  float64   `json:"precio_base"`
	Inicio      time.Time `json:"inicio"`
	Cierre      time.Time `json:"cierre"`
}

type auctionView struct {
	ID            string               `json:"id"`
	VendedorID    string               `json:"vendedor_id,omitempty"`
	Titulo        string               `json:"titulo"`
	Descripcion   string               `json:"descripcion"`
	PrecioBase    float64              `json:"precio_base"`
	PrecioActual  float64              `json:"precio_actual"`
	PostorLiderID string               `json:"postor_lider_id,omitempty"`
	Estado        domain.EstadoSubasta `json:"estado"`
	Inicio        time.Time            `json:"inicio"`
	Cierre        time.Time            `json:"cierre"`
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
		ID: "sub_" + hex.EncodeToString(idBytes), VendedorID: user.ID,
		Titulo: request.Titulo, Descripcion: request.Descripcion,
		PrecioBase: request.PrecioBase, Inicio: request.Inicio, Cierre: request.Cierre,
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

func (h *AuctionHandler) Close(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.catalog.Cerrar)
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
		ID: subasta.ID, VendedorID: subasta.VendedorID, Titulo: subasta.Titulo,
		Descripcion: subasta.Descripcion, PrecioBase: subasta.PrecioBase,
		PrecioActual: subasta.PrecioActual, PostorLiderID: subasta.PostorLiderID,
		Estado: subasta.Estado, Inicio: subasta.FechaInicio, Cierre: subasta.FechaCierre,
	}
}

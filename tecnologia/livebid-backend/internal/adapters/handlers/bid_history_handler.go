package handlers

import (
	"errors"
	"net/http"
	"strconv"

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
	query, err := parseBidHistoryQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bid history query"})
		return
	}
	result, err := h.service.ListPage(r.Context(), r.PathValue("id"), query)
	if errors.Is(err, domain.ErrSubastaNoEncontrada) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auction not found"})
		return
	}
	if err != nil {
		if errors.Is(err, services.ErrInvalidBidHistoryQuery) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bid history query"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseBidHistoryQuery(r *http.Request) (services.BidHistoryQuery, error) {
	query := services.BidHistoryQuery{Page: 1, PerPage: 50}
	values := r.URL.Query()
	query.BidderID = values.Get("postor_id")
	for name, target := range map[string]*int{"page": &query.Page, "per_page": &query.PerPage} {
		value := values.Get(name)
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return services.BidHistoryQuery{}, err
		}
		*target = parsed
	}
	if value := values.Get("monto_minimo"); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return services.BidHistoryQuery{}, err
		}
		query.MinimumAmount = parsed
	}
	return query, nil
}

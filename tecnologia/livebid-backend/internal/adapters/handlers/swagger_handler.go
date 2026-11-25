package handlers

import (
	"net/http"

	"github.com/fxrl01/livebid-backend/internal/adapters/docs"
)

// SwaggerHandler expone el contrato OpenAPI 3.0.3 (YAML/JSON) y la interfaz Swagger UI embebida.
type SwaggerHandler struct {
	yamlSpec []byte
	jsonSpec []byte
	uiHTML   []byte
}

// NewSwaggerHandler construye un handler de documentacion usando los artefactos embebidos.
func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{
		yamlSpec: docs.OpenAPIYAML,
		jsonSpec: docs.OpenAPIJSON,
		uiHTML:   docs.SwaggerHTML,
	}
}

// RegisterSwaggerRoutes registra los endpoints publicos de Swagger UI y OpenAPI en el mux dado.
func RegisterSwaggerRoutes(mux *http.ServeMux) {
	h := NewSwaggerHandler()
	mux.HandleFunc("GET /swagger", h.ServeUI)
	mux.HandleFunc("GET /swagger/{$}", h.ServeUI)
	mux.HandleFunc("GET /docs", h.ServeUI)
	mux.HandleFunc("GET /docs/{$}", h.ServeUI)
	mux.HandleFunc("GET /api/v1/docs", h.ServeUI)
	mux.HandleFunc("GET /api/v1/docs/{$}", h.ServeUI)

	mux.HandleFunc("GET /openapi.yaml", h.ServeSpecYAML)
	mux.HandleFunc("GET /api/v1/openapi.yaml", h.ServeSpecYAML)

	mux.HandleFunc("GET /openapi.json", h.ServeSpecJSON)
	mux.HandleFunc("GET /api/v1/openapi.json", h.ServeSpecJSON)
}

// ServeUI entrega la pagina interactiva Swagger UI en HTML.
func (h *SwaggerHandler) ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.uiHTML)
}

// ServeSpecYAML entrega la especificacion canonica OpenAPI 3.0.3 en formato YAML.
func (h *SwaggerHandler) ServeSpecYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.yamlSpec)
}

// ServeSpecJSON entrega la especificacion OpenAPI 3.0.3 compilada en formato JSON.
func (h *SwaggerHandler) ServeSpecJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.jsonSpec)
}

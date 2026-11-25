package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fxrl01/livebid-backend/internal/adapters/docs"
)

func TestSwaggerHandlerEndpointsAndContentTypes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSwaggerRoutes(mux)

	uiPaths := []string{"/swagger", "/swagger/", "/docs", "/docs/", "/api/v1/docs", "/api/v1/docs/"}
	for _, path := range uiPaths {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200, got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("GET %s expected text/html Content-Type, got %q", path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "SwaggerUIBundle") || !strings.Contains(body, "/api/v1/openapi.json") {
			t.Fatalf("GET %s missing SwaggerUIBundle initialization or spec url", path)
		}
	}

	yamlPaths := []string{"/openapi.yaml", "/api/v1/openapi.yaml"}
	for _, path := range yamlPaths {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200, got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/yaml") {
			t.Fatalf("GET %s expected application/yaml, got %q", path, ct)
		}
		if !strings.Contains(rec.Body.String(), "openapi: 3.0.3") {
			t.Fatalf("GET %s missing openapi: 3.0.3 header", path)
		}
	}

	jsonPaths := []string{"/openapi.json", "/api/v1/openapi.json"}
	for _, path := range jsonPaths {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200, got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("GET %s expected application/json, got %q", path, ct)
		}
	}
}

func TestOpenAPISpecMatchesMainServerRoutesAndRefs(t *testing.T) {
	var spec struct {
		OpenAPI    string                               `json:"openapi"`
		Paths      map[string]map[string]map[string]any `json:"paths"`
		Components map[string]map[string]any            `json:"components"`
	}
	if err := json.Unmarshal(docs.OpenAPIJSON, &spec); err != nil {
		t.Fatalf("unmarshal embedded openapi.json: %v", err)
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("expected OpenAPI version 3.0.3, got %q", spec.OpenAPI)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("expected non-empty OpenAPI paths")
	}

	// Verify unique operationIds across all operations
	seenOps := make(map[string]string)
	for path, methods := range spec.Paths {
		for method, op := range methods {
			opID, _ := op["operationId"].(string)
			if opID == "" {
				t.Fatalf("%s %s missing operationId", strings.ToUpper(method), path)
			}
			if prev, dup := seenOps[opID]; dup {
				t.Fatalf("duplicate operationId %q on %s %s and %s", opID, strings.ToUpper(method), path, prev)
			}
			seenOps[opID] = strings.ToUpper(method) + " " + path
		}
	}

	// Verify every local $ref resolves to an existing component
	var rawSpec any
	if err := json.Unmarshal(docs.OpenAPIJSON, &rawSpec); err != nil {
		t.Fatalf("re-unmarshal raw spec: %v", err)
	}
	checkLocalRefs(t, rawSpec, spec.Components)

	// Verify 100% parity with API & WS routes registered in cmd/server/main.go
	mainPath := filepath.Join("..", "..", "..", "cmd", "server", "main.go")
	mainSrc, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read cmd/server/main.go: %v", err)
	}
	routePattern := regexp.MustCompile(`mux\.Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE)\s+(/[^"\s]+)"`)
	matches := routePattern.FindAllStringSubmatch(string(mainSrc), -1)
	if len(matches) == 0 {
		t.Fatal("failed to parse routes from cmd/server/main.go")
	}
	for _, m := range matches {
		method := strings.ToLower(m[1])
		path := m[2]
		if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws/") && path != "/health" {
			continue
		}
		pathItem, ok := spec.Paths[path]
		if !ok {
			t.Fatalf("route %s %s in cmd/server/main.go is missing from openapi.yaml/json", strings.ToUpper(method), path)
		}
		if _, ok := pathItem[method]; !ok {
			t.Fatalf("method %s for path %s in cmd/server/main.go is missing from openapi.yaml/json", strings.ToUpper(method), path)
		}
	}
}

func checkLocalRefs(t *testing.T, node any, components map[string]map[string]any) {
	t.Helper()
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			const prefix = "#/components/"
			if !strings.HasPrefix(ref, prefix) {
				t.Fatalf("unsupported external/invalid $ref %q", ref)
			}
			parts := strings.Split(strings.TrimPrefix(ref, prefix), "/")
			if len(parts) != 2 {
				t.Fatalf("unexpected $ref structure %q", ref)
			}
			group, exists := components[parts[0]]
			if !exists {
				t.Fatalf("$ref %q references missing component group %q", ref, parts[0])
			}
			if _, ok := group[parts[1]]; !ok {
				t.Fatalf("$ref %q references missing component %q", ref, parts[1])
			}
		}
		for _, child := range v {
			checkLocalRefs(t, child, components)
		}
	case []any:
		for _, item := range v {
			checkLocalRefs(t, item, components)
		}
	}
}

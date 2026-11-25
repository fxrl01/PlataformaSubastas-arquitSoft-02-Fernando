package docs

import _ "embed"

// OpenAPIYAML contiene la especificacion canonica OpenAPI 3.0.3 de LiveBid en formato YAML.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte

// OpenAPIJSON contiene la especificacion OpenAPI 3.0.3 compilada a JSON para Swagger UI y herramientas automatizadas.
//
//go:embed openapi.json
var OpenAPIJSON []byte

// SwaggerHTML contiene la interfaz interactiva Swagger UI servida por el motor HTTP de LiveBid.
//
//go:embed swagger.html
var SwaggerHTML []byte

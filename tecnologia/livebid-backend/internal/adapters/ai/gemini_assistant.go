package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AssistantTurn representa un mensaje previo en la conversación con el asistente.
type AssistantTurn struct {
	Role string `json:"rol"`
	Text string `json:"texto"`
}

// AssistantAuctionContext resume los datos reales de la subasta que el usuario está viendo.
type AssistantAuctionContext struct {
	ID              string  `json:"id"`
	Title           string  `json:"titulo"`
	Category        string  `json:"categoria"`
	Status          string  `json:"estado"`
	BasePrice       float64 `json:"precio_base"`
	CurrentPrice    float64 `json:"precio_actual"`
	NextMinimumBid  float64 `json:"siguiente_oferta_minima"`
	RequiredDeposit float64 `json:"garantia_estimada"`
	TotalCatalog    int     `json:"total_lotes_catalogo"`
}

// AssistantQuery encapsula la consulta del usuario y su contexto operativo en la app.
type AssistantQuery struct {
	Message        string                  `json:"mensaje"`
	UserRole       string                  `json:"rol_usuario,omitempty"`
	UserName       string                  `json:"nombre_usuario,omitempty"`
	HasGuarantee   bool                    `json:"tiene_garantia,omitempty"`
	AuctionContext AssistantAuctionContext `json:"contexto_subasta,omitempty"`
	History        []AssistantTurn         `json:"historial,omitempty"`
}

// AssistantReply contiene la respuesta generada por Google Gemini o el motor contextual de respaldo.
type AssistantReply struct {
	Reply      string   `json:"respuesta"`
	Provider   string   `json:"proveedor"`
	Model      string   `json:"modelo"`
	LocalMode  bool     `json:"modo_local"`
	Suggested  []string `json:"sugerencias,omitempty"`
}

// GeminiAssistant conecta el chat de asistencia de la plataforma con Google Gemini API
// y provee respuestas contextuales deterministas cuando no se ha configurado GEMINI_API_KEY.
type GeminiAssistant struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewGeminiAssistant(apiKey string) *GeminiAssistant {
	return &GeminiAssistant{
		apiKey:     strings.TrimSpace(apiKey),
		model:      "gemini-2.5-flash",
		baseURL:    "https://generativelanguage.googleapis.com/v1beta",
		httpClient: &http.Client{Timeout: 6 * time.Second},
	}
}

func (a *GeminiAssistant) WithBaseURL(baseURL string) *GeminiAssistant {
	if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
		a.baseURL = trimmed
	}
	return a
}

func (a *GeminiAssistant) isRealKeyConfigured() bool {
	if a.apiKey == "" {
		return false
	}
	lower := strings.ToLower(a.apiKey)
	return !strings.HasPrefix(lower, "replace-") && !strings.Contains(lower, "replace_with")
}

func (a *GeminiAssistant) Answer(ctx context.Context, q AssistantQuery) (AssistantReply, error) {
	msg := strings.TrimSpace(q.Message)
	if msg == "" {
		return AssistantReply{}, fmt.Errorf("empty assistant message")
	}

	if a.isRealKeyConfigured() {
		if reply, err := a.callGeminiAPI(ctx, q); err == nil && strings.TrimSpace(reply) != "" {
			return AssistantReply{
				Reply:     reply,
				Provider:  "google-gemini",
				Model:     a.model,
				LocalMode: false,
				Suggested: defaultSuggestionsForRole(q.UserRole),
			}, nil
		}
	}

	return AssistantReply{
		Reply:     buildContextualGuideReply(q),
		Provider:  "aureum-contextual-assistant",
		Model:     "gemini-ready-local-guide",
		LocalMode: true,
		Suggested: defaultSuggestionsForRole(q.UserRole),
	}, nil
}

func (a *GeminiAssistant) callGeminiAPI(ctx context.Context, q AssistantQuery) (string, error) {
	sysInstruction := buildSystemPrompt(q)

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	type payload struct {
		SystemInstruction *content       `json:"systemInstruction,omitempty"`
		Contents          []content      `json:"contents"`
		GenerationConfig  map[string]any `json:"generationConfig,omitempty"`
	}

	var contents []content
	start := 0
	if len(q.History) > 6 {
		start = len(q.History) - 6
	}
	for _, turn := range q.History[start:] {
		txt := strings.TrimSpace(turn.Text)
		if txt == "" {
			continue
		}
		role := "user"
		if strings.EqualFold(turn.Role, "assistant") || strings.EqualFold(turn.Role, "model") {
			role = "model"
		}
		contents = append(contents, content{Role: role, Parts: []part{{Text: txt}}})
	}
	contents = append(contents, content{Role: "user", Parts: []part{{Text: strings.TrimSpace(q.Message)}}})

	reqBody, err := json.Marshal(payload{
		SystemInstruction: &content{Parts: []part{{Text: sysInstruction}}},
		Contents:          contents,
		GenerationConfig: map[string]any{
			"temperature":     0.35,
			"maxOutputTokens": 420,
		},
	})
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", a.baseURL, a.model, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty gemini candidates")
	}
	return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), nil
}

func buildSystemPrompt(q AssistantQuery) string {
	role := q.UserRole
	if role == "" {
		role = "VISITANTE"
	}
	ac := q.AuctionContext
	return fmt.Sprintf(`Eres el Concierge y Asistente Inteligente de Aureum & Co. (LiveBid), una casa de subastas en tiempo real.
Responde siempre en español claro, amable, breve y útil (máximo 4 párrafos cortos o viñetas), SIN usar jerga interna de programación como JWT, PostgreSQL, JSONB o RBAC.

Funciones principales de la plataforma que debes explicar cuando te pregunten:
1. Registro y verificación: cualquier persona crea su cuenta eligiendo perfil Comprador/Postor, Vendedor/Consignante o Martillero. Un administrador verifica su identidad para activar la cuenta.
2. Publicar bienes y objetos de valor: los usuarios con perfil Vendedor, Martillero o Administrador usan el botón "Publicar Bien" para subir fotografías reales desde su dispositivo o enlace, título, categoría, ubicación, precio base, precio de reserva y tasador.
3. Depósito de garantía reembolsable: antes de ofertar en un lote, el comprador autoriza una retención temporal con tarjeta (10%% del precio base, o 20%% según evaluación de riesgo). Si no gana la subasta, el dinero se libera automáticamente sin cargo.
4. Ofertas en vivo y cierre justo (Anti-Sniping): las ofertas se actualizan al instante en pantalla. Si entra una oferta en los últimos 15 segundos, el reloj suma automáticamente +30 segundos para dar oportunidad justa a todos.
5. Certificación transparente: cada oferta queda sellada digitalmente para garantizar que nadie altere el orden ni los montos.

Contexto actual del usuario:
- Nombre: %s | Perfil: %s | Garantía activa en este lote: %t
- Lote en pantalla: %s (%s) | Estado: %s | Precio base: S/. %.0f | Oferta actual: S/. %.0f | Siguiente oferta mínima: S/. %.0f | Garantía estimada: S/. %.0f | Total lotes en catálogo: %d`,
		q.UserName, role, q.HasGuarantee,
		ac.Title, ac.ID, ac.Status, ac.BasePrice, ac.CurrentPrice, ac.NextMinimumBid, ac.RequiredDeposit, ac.TotalCatalog)
}

func buildContextualGuideReply(q AssistantQuery) string {
	lower := strings.ToLower(q.Message)
	ac := q.AuctionContext
	loteNombre := ac.Title
	if loteNombre == "" {
		loteNombre = "el lote actual en sala"
	}
	nextMin := ac.NextMinimumBid
	if nextMin <= 0 && ac.CurrentPrice > 0 {
		nextMin = ac.CurrentPrice + 1000
	}
	deposito := ac.RequiredDeposit
	if deposito <= 0 && ac.BasePrice > 0 {
		deposito = ac.BasePrice * 0.10
	}

	switch {
	case strings.Contains(lower, "vender") || strings.Contains(lower, "subir") || strings.Contains(lower, "publicar") || strings.Contains(lower, "producto") || strings.Contains(lower, "bien") || strings.Contains(lower, "lote"):
		return fmt.Sprintf("Para publicar un bien u objeto de valor en subasta sigue estos pasos:\n\n1. **Cuenta de Vendedor:** Crea tu cuenta eligiendo el perfil **Vendedor / Consignante** (una vez verificada tu identidad quedará activa).\n2. **Publicar Bien:** Haz clic en el botón **«Publicar Bien»** en la parte superior o en la sección de catálogo.\n3. **Fotografía y Ficha:** Sube la foto desde tu celular o computadora, indica el título, categoría (vehículos, relojería, joyas, arte o inmuebles), precio base de apertura, ubicación y duración.\n\nAl guardar, el bien aparece inmediatamente en la **Colección de Bienes en Subasta** (actualmente hay **%d lote(s)** publicados) listo para recibir ofertas en vivo.", maxInt(ac.TotalCatalog, 1))

	case strings.Contains(lower, "garant") || strings.Contains(lower, "deposito") || strings.Contains(lower, "depósito") || strings.Contains(lower, "tarjeta") || strings.Contains(lower, "reembols"):
		estadoGarantia := "Aún no tienes una garantía activa en este lote."
		if q.HasGuarantee {
			estadoGarantia = "Ya cuentas con tu garantía activa y habilitada para ofertar en este lote."
		}
		return fmt.Sprintf("El **depósito de garantía reembolsable** asegura que todas las ofertas en sala sean serias:\n\n• Para **%s**, la garantía estimada es de **S/. %.0f** (10%% del precio base de S/. %.0f).\n• **No es un cobro inmediato:** solo se congela temporalmente el cupo en tu tarjeta mediante pasarela segura.\n• Si otro participante gana la subasta, **la retención se libera automáticamente** sin ningún costo para ti.\n• %s", loteNombre, deposito, ac.BasePrice, estadoGarantia)

	case strings.Contains(lower, "pujar") || strings.Contains(lower, "ofertar") || strings.Contains(lower, "oferta") || strings.Contains(lower, "cuanto") || strings.Contains(lower, "cuánto") || strings.Contains(lower, "precio"):
		return fmt.Sprintf("Sobre **%s** (%s):\n\n• **Oferta máxima actual:** S/. %.0f\n• **Siguiente oferta mínima sugerida:** S/. %.0f\n• **Estado de la sala:** %s\n\nPara ofertar: inicia sesión, activa tu depósito de garantía reembolsable (S/. %.0f) y usa los botones de incremento rápido o escribe tu monto y pulsa **«Realizar Oferta»**.", loteNombre, ac.ID, ac.CurrentPrice, nextMin, ac.Status, deposito)

	case strings.Contains(lower, "15") || strings.Contains(lower, "tiempo") || strings.Contains(lower, "reloj") || strings.Contains(lower, "sniping") || strings.Contains(lower, "ultimo") || strings.Contains(lower, "último"):
		return "Contamos con la **Regla de Cierre Justo (Anti-Sniping)**:\n\nSi un participante realiza una oferta válida cuando quedan **15 segundos o menos** en el cronómetro, la sala añade automáticamente **+30 segundos adicionales**. Así nadie pierde un artículo por un clic en el último segundo y todos tienen tiempo real de responder como en un salón presencial."

	case strings.Contains(lower, "registr") || strings.Contains(lower, "cuenta") || strings.Contains(lower, "rol") || strings.Contains(lower, "iniciar") || strings.Contains(lower, "aprobar"):
		return "Crear y usar tu cuenta es muy sencillo:\n\n1. Pulsa **«Ingresar / Crear Cuenta»** arriba a la derecha.\n2. En **Crear Cuenta**, completa tu nombre, correo, documento de identidad y elige cómo deseas participar: **Comprador (Postor)**, **Vendedor (Consignante)** o **Martillero**.\n3. Tras la revisión de seguridad del administrador tu cuenta pasa a **Activa**, permitiéndote ofertar en vivo o publicar tus propios bienes."

	default:
		saludo := "Hola"
		if q.UserName != "" && !strings.EqualFold(q.UserName, "Visitante") {
			saludo = "Hola, " + q.UserName
		}
		return fmt.Sprintf("%s. Soy tu asistente de **Aureum & Co. LiveBid**. Estoy viendo contigo **%s** (oferta actual **S/. %.0f**, siguiente mínima **S/. %.0f**).\n\nPuedo ayudarte con:\n• **Comprar y ofertar en vivo:** cuánto ofertar y cómo funciona el reloj de cierre justo (+30 s).\n• **Depósito reembolsable:** cómo activar tu garantía con tarjeta sin cobros si no ganas.\n• **Vender bienes de valor:** cómo registrar una cuenta de Vendedor y subir tus propios lotes con fotografías.", saludo, loteNombre, ac.CurrentPrice, nextMin)
	}
}

func defaultSuggestionsForRole(role string) []string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "VENDEDOR", "MARTILLERO", "ADMIN":
		return []string{
			"¿Cómo publico un nuevo bien con fotos?",
			"¿Cuánto es la oferta mínima en este lote?",
			"¿Cómo funciona el cierre justo de +30 s?",
		}
	default:
		return []string{
			"¿Cuánto necesito para ofertar en este lote?",
			"¿Cómo funciona el depósito reembolsable?",
			"¿Cómo puedo vender un bien de valor?",
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

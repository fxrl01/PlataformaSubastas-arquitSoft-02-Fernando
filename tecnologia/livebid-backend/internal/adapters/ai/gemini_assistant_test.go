package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiAssistantLocalContextualModesAndRemoteAPI(t *testing.T) {
	local := NewGeminiAssistant("replace-with-provider-key")
	ctx := AssistantAuctionContext{
		ID:              "SUB-2026-001",
		Title:           "Reloj Patek Philippe Nautilus",
		Status:          "ACTIVA",
		BasePrice:       40000,
		CurrentPrice:    42000,
		NextMinimumBid:  43000,
		RequiredDeposit: 4000,
		TotalCatalog:    3,
	}

	cases := []string{
		"¿Cómo puedo subir y vender un bien?",
		"¿Cómo funciona el depósito de garantía reembolsable?",
		"¿Cuánto necesito para pujar en este lote?",
		"¿Qué pasa en los últimos 15 segundos del reloj?",
		"¿Cómo me registro en la plataforma?",
		"Hola, ayúdame a entender la sala",
	}
	for _, msg := range cases {
		res, err := local.Answer(context.Background(), AssistantQuery{
			Message:        msg,
			UserName:       "Elena",
			UserRole:       "POSTOR",
			HasGuarantee:   true,
			AuctionContext: ctx,
		})
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", msg, err)
		}
		if !res.LocalMode || strings.TrimSpace(res.Reply) == "" || len(res.Suggested) == 0 {
			t.Fatalf("expected non-empty local contextual reply for %q, got %+v", msg, res)
		}
	}

	if _, err := local.Answer(context.Background(), AssistantQuery{Message: "   "}); err == nil {
		t.Fatal("expected error on empty message")
	}

	// Verificar integración HTTP real con servidor simulado de Gemini API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Respuesta generada por Gemini 2.5 Flash para tu lote."}]}}]}`))
	}))
	defer srv.Close()

	remote := NewGeminiAssistant("AIzaSyRealTestKey123456").WithBaseURL(srv.URL)
	reply, err := remote.Answer(context.Background(), AssistantQuery{
		Message:        "Explícame cómo ofertar",
		UserRole:       "VENDEDOR",
		AuctionContext: ctx,
		History: []AssistantTurn{
			{Role: "user", Text: "Hola"},
			{Role: "assistant", Text: "Bienvenido"},
		},
	})
	if err != nil {
		t.Fatalf("expected remote gemini success, got %v", err)
	}
	if reply.LocalMode || reply.Provider != "google-gemini" || !strings.Contains(reply.Reply, "Gemini 2.5 Flash") {
		t.Fatalf("unexpected remote reply: %+v", reply)
	}
}

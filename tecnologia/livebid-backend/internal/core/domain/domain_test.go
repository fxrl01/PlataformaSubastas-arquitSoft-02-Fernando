package domain_test

import (
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

func TestCalcularIncrementoMinimoPorTramos(t *testing.T) {
	cases := []struct {
		price    float64
		expected float64
	}{
		{500, 50},
		{1_000, 100},
		{9_999, 100},
		{10_000, 500},
		{45_000, 500},
		{50_000, 1_000},
		{120_000, 2_500},
	}

	for _, tc := range cases {
		got := domain.CalcularIncrementoMinimo(tc.price)
		if got != tc.expected {
			t.Fatalf("CalcularIncrementoMinimo(%.2f) = %.2f; want %.2f", tc.price, got, tc.expected)
		}
	}
}

func TestSubastaLifecycleAntiSnipingAndSelfOutbidPrevention(t *testing.T) {
	now := time.Now()
	closeAt := now.Add(10 * time.Minute)
	subasta := domain.NuevaSubasta("SUB-DOM-01", "Camioneta 4x4", "Lote impecable", 20000, now, closeAt)

	if subasta.Estado != domain.SubastaPendiente {
		t.Fatalf("expected initial state PENDIENTE, got %s", subasta.Estado)
	}
	if err := subasta.Activar(); err != nil {
		t.Fatalf("Activar failed: %v", err)
	}
	if err := subasta.IniciarConteo(); err != nil {
		t.Fatalf("IniciarConteo failed: %v", err)
	}
	if subasta.Estado != domain.SubastaEnConteo {
		t.Fatalf("expected EN_CONTEO, got %s", subasta.Estado)
	}

	// Puja dentro de los últimos 10s antes del cierre dispara Anti-Sniping (+30s)
	snipingTime := closeAt.Add(-10 * time.Second)
	_, extended, err := subasta.ValidarYRegistrarPuja("puja-1", "postor-A", 20500, snipingTime)
	if err != nil {
		t.Fatalf("ValidarYRegistrarPuja failed: %v", err)
	}
	if !extended || !subasta.FechaCierre.Equal(closeAt.Add(domain.ExtensionAntiSniping)) {
		t.Fatalf("expected anti-sniping +30s extension, got extended=%v close=%v", extended, subasta.FechaCierre)
	}

	// El mismo postor líder no puede auto-superarse consecutivamente
	if _, _, err := subasta.ValidarYRegistrarPuja("puja-2", "postor-A", 21500, snipingTime.Add(time.Second)); err != domain.ErrPostorIgual {
		t.Fatalf("expected ErrPostorIgual on self-outbid, got %v", err)
	}

	if err := subasta.Cerrar(); err != nil {
		t.Fatalf("Cerrar failed: %v", err)
	}
	if subasta.Estado != domain.SubastaCerrada {
		t.Fatalf("expected CERRADA, got %s", subasta.Estado)
	}
}

func TestMerkleTreeProofAndWalletValidation(t *testing.T) {
	if !domain.IsValidWalletAddress("0x71C7656EC7ab88b098defB751B7401B5f6d8976F") {
		t.Fatal("expected valid EVM hex address to pass")
	}
	if domain.IsValidWalletAddress("invalid-wallet") {
		t.Fatal("expected invalid wallet address to fail")
	}

	leaves := []string{"a1b2c3", "d4e5f6", "7890ab"}
	root := domain.ComputeMerkleRoot(leaves)
	for idx, leaf := range leaves {
		proof, ok := domain.GenerateMerkleProof(leaves, idx)
		if !ok || !domain.VerifyMerkleProof(leaf, root, proof) {
			t.Fatalf("expected valid Merkle proof at index %d", idx)
		}
	}
}

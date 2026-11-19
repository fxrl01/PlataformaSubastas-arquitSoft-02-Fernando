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
	if emptyRoot := domain.ComputeMerkleRoot(nil); emptyRoot == "" {
		t.Fatal("expected deterministic non-empty root for empty leaves")
	}
	if _, ok := domain.GenerateMerkleProof(leaves, 99); ok {
		t.Fatal("expected out-of-range index to fail proof generation")
	}
	if domain.VerifyMerkleProof("", root, nil) {
		t.Fatal("expected empty leaf hash to fail verification")
	}
}

func TestUserValidationRolesStatusesAndAuctionHelpers(t *testing.T) {
	now := time.Now()
	if _, err := domain.NewUser("u1", "Ana", "bad-email", "hash", now); err != domain.ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
	if _, err := domain.NewUser("u1", "", "ana@livebid.pe", "hash", now); err != domain.ErrInvalidUser {
		t.Fatalf("expected ErrInvalidUser, got %v", err)
	}
	user, err := domain.NewUser("u1", "Ana", "ANA@livebid.pe ", "hash", now)
	if err != nil || user.Email != "ana@livebid.pe" || user.Role != domain.RoleBidder || user.Status != domain.UserPending {
		t.Fatalf("unexpected user: %+v err=%v", user, err)
	}
	if !domain.IsValidRole(domain.RoleAdmin) || domain.IsValidRole("SUPERUSER") {
		t.Fatal("unexpected IsValidRole behavior")
	}
	if !domain.IsValidUserStatus(domain.UserBlocked) || domain.IsValidUserStatus("UNKNOWN") {
		t.Fatal("unexpected IsValidUserStatus behavior")
	}

	sub := domain.NuevaSubasta("s1", "Lote", "Desc", 1000, now, now.Add(time.Hour))
	if sub.TienePujas() || sub.UltimaPuja() != nil {
		t.Fatal("expected empty bids initially")
	}
	if err := sub.AsignarEncargado(domain.EncargadoSubasta{}); err != domain.ErrSubastaInvalida {
		t.Fatalf("expected ErrSubastaInvalida on empty encargado, got %v", err)
	}
	if err := sub.AsignarEncargado(domain.EncargadoSubasta{MartilleroID: "M1", Nombre: "Martillero"}); err != nil || sub.MartilleroID != "M1" {
		t.Fatalf("unexpected AsignarEncargado result: %v", err)
	}
	if err := sub.ActualizarDetallesLote(domain.DetallesLote{PrecioReserva: -10}); err != domain.ErrSubastaInvalida {
		t.Fatalf("expected ErrSubastaInvalida on negative reserve price, got %v", err)
	}
	if err := sub.ActualizarDetallesLote(domain.DetallesLote{Categoria: "ARTE", PrecioReserva: 1200}); err != nil {
		t.Fatalf("unexpected ActualizarDetallesLote error: %v", err)
	}
	if sub.Estado.String() != "PENDIENTE" {
		t.Fatalf("unexpected state string: %s", sub.Estado.String())
	}
	_ = sub.Activar()
	if _, _, err := sub.ValidarYRegistrarPuja("p1", "b1", 1100, now.Add(time.Minute)); err != nil {
		t.Fatalf("unexpected bid error: %v", err)
	}
	if !sub.TienePujas() || sub.UltimaPuja() == nil || sub.UltimaPuja().ID != "p1" {
		t.Fatal("expected UltimaPuja to return p1")
	}
	if _, _, err := sub.ValidarYRegistrarPuja("p2", "b2", 1120, now.Add(2*time.Minute)); err != domain.ErrPujaInferior {
		t.Fatalf("expected ErrPujaInferior, got %v", err)
	}

	// AgentMandate limits & expiration
	mandate := &domain.AgentMandate{
		ID:                  "m1",
		MaxAuthorizedAmount: 1500,
		ExpiresAt:           now.Add(time.Hour),
	}
	if err := mandate.CanAuthorizeBid(1500, now); err != nil {
		t.Fatalf("expected valid mandate authorization, got %v", err)
	}
	if err := mandate.CanAuthorizeBid(1600, now); err != domain.ErrAgentMandateExceeded {
		t.Fatalf("expected ErrAgentMandateExceeded, got %v", err)
	}
	if err := mandate.CanAuthorizeBid(1000, now.Add(2*time.Hour)); err != domain.ErrAgentMandateExpired {
		t.Fatalf("expected ErrAgentMandateExpired, got %v", err)
	}
	if domain.RequiredGuaranteeWithRatio(1000, 0) != 100 || domain.RequiredGuaranteeWithRatio(1000, 0.2) != 200 {
		t.Fatal("unexpected guarantee ratio calculation")
	}
}

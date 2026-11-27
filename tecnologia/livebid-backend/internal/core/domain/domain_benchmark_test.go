package domain

import (
	"fmt"
	"testing"
	"time"
)

func TestAntiSnipingExact15SecondBoundary(t *testing.T) {
	inicio := time.Date(2026, 11, 27, 15, 0, 0, 0, time.UTC)
	cierre := inicio.Add(60 * time.Second)
	sub := NuevaSubasta("SUB-BOUNDARY", "Lote Exacto", "Prueba borde 15s", 1000, inicio, cierre)
	if err := sub.Activar(); err != nil {
		t.Fatalf("activar: %v", err)
	}

	// A los 15s + 1ms restantes NO debe activar Anti-Sniping
	tFuera := cierre.Add(-(15*time.Second + time.Millisecond))
	_, ext1, err := sub.ValidarYRegistrarPuja("p1", "usr_1", 1100, tFuera)
	if err != nil || ext1 {
		t.Fatalf("expected valid bid without anti-sniping at 15.001s, got ext=%t err=%v", ext1, err)
	}

	// Exactamente a los 15.000s restantes SÍ debe activar Anti-Sniping (+30s)
	tExacto := cierre.Add(-15 * time.Second)
	_, ext2, err := sub.ValidarYRegistrarPuja("p2", "usr_2", 1200, tExacto)
	if err != nil || !ext2 {
		t.Fatalf("expected anti-sniping extension at exact 15.000s, got ext=%t err=%v", ext2, err)
	}
	if !sub.FechaCierre.Equal(cierre.Add(ExtensionAntiSniping)) {
		t.Fatalf("expected FechaCierre extended by 30s, got %v", sub.FechaCierre)
	}
}

func BenchmarkValidarYRegistrarPuja(b *testing.B) {
	inicio := time.Now()
	cierre := inicio.Add(24 * time.Hour)
	sub := NuevaSubasta("SUB-BENCH", "Lote Rendimiento", "Benchmark CPU", 1000, inicio, cierre)
	_ = sub.Activar()

	monto := 2000.0
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		bidder := "postor_a"
		if i%2 == 1 {
			bidder = "postor_b"
		}
		monto += 5000
		_, _, err := sub.ValidarYRegistrarPuja("puja_bench", bidder, monto, inicio)
		if err != nil {
			b.Fatalf("unexpected error at iter %d: %v", i, err)
		}
	}
}

func BenchmarkComputeMerkleRoot512Bids(b *testing.B) {
	hashes := make([]string, 512)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("%064x", i+1)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		root := ComputeMerkleRoot(hashes)
		if len(root) != 66 {
			b.Fatalf("unexpected merkle root len: %d", len(root))
		}
	}
}

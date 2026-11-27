package realtime

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type benchPubSubStream struct {
	ch     chan []byte
	closed bool
	mu     sync.Mutex
}

func newBenchPubSubStream(buf int) *benchPubSubStream {
	return &benchPubSubStream{ch: make(chan []byte, buf)}
}

func (f *benchPubSubStream) Messages() <-chan []byte {
	return f.ch
}

func (f *benchPubSubStream) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.ch)
	}
	return nil
}

func (f *benchPubSubStream) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func TestAuctionRoomHubSlowClientDoesNotBlockFastClientsAndCleansUp(t *testing.T) {
	streams := make(map[string]*benchPubSubStream)
	var mu sync.Mutex

	hub := NewAuctionRoomHub(func(_ context.Context, channel string) (PubSubStream, error) {
		s := newBenchPubSubStream(128)
		mu.Lock()
		streams[channel] = s
		mu.Unlock()
		return s, nil
	})

	slowCh, leaveSlow, err := hub.Join("SUB-HUB-1")
	if err != nil {
		t.Fatalf("join slow client: %v", err)
	}
	fastCh, leaveFast, err := hub.Join("SUB-HUB-1")
	if err != nil {
		t.Fatalf("join fast client: %v", err)
	}

	if hub.ActiveRooms() != 1 {
		t.Fatalf("expected 1 active room, got %d", hub.ActiveRooms())
	}

	mu.Lock()
	stream := streams[RedisChannelPrefix()+"SUB-HUB-1"]
	mu.Unlock()

	// Enviar 45 eventos seguidos: el cliente lento (buffer 32) nunca debe bloquear al cliente rápido
	for i := 1; i <= 45; i++ {
		stream.ch <- []byte(fmt.Sprintf(`{"seq":%d}`, i))
		select {
		case got := <-fastCh:
			if len(got) == 0 {
				t.Fatal("empty payload on fast client")
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("fast client timed out at message %d due to slow client blocking", i)
		}
	}

	// El cliente lento debe conservar los mensajes más recientes sin bloquearse
	if len(slowCh) == 0 {
		t.Fatal("expected buffered messages in slow client channel")
	}

	// Idempotencia de leave() y cierre automático de la sala al salir el último espectador
	leaveSlow()
	leaveSlow()
	if hub.ActiveRooms() != 1 {
		t.Fatalf("room should stay open while fast client remains, got %d", hub.ActiveRooms())
	}

	leaveFast()
	if hub.ActiveRooms() != 0 {
		t.Fatalf("expected 0 active rooms after all clients left, got %d", hub.ActiveRooms())
	}
	if !stream.isClosed() {
		t.Fatal("expected underlying PubSubStream to be closed when room empties")
	}
}

func BenchmarkRoomHubFanOut500Clients(b *testing.B) {
	stream := newBenchPubSubStream(4096)
	hub := NewAuctionRoomHub(func(_ context.Context, _ string) (PubSubStream, error) {
		return stream, nil
	})

	const numClients = 500
	leaves := make([]func(), 0, numClients)
	channels := make([]<-chan []byte, 0, numClients)
	for i := 0; i < numClients; i++ {
		ch, leave, err := hub.Join("SUB-BENCH")
		if err != nil {
			b.Fatalf("join %d: %v", i, err)
		}
		channels = append(channels, ch)
		leaves = append(leaves, leave)
	}
	defer func() {
		for _, lv := range leaves {
			lv()
		}
	}()

	payload := []byte(`{"event":"bid.accepted","monto":52000}`)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		stream.ch <- payload
		<-channels[0]
	}
}

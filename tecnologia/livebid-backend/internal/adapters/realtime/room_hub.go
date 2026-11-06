package realtime

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

// PubSubStream abstrae un canal de suscripción para facilitar pruebas unitarias del Hub sin red.
type PubSubStream interface {
	Messages() <-chan []byte
	Close() error
}

// PubSubSubscriber abre una única suscripción hacia el broker por sala.
type PubSubSubscriber func(ctx context.Context, channel string) (PubSubStream, error)

type redisPubSubStream struct {
	pubsub *redis.PubSub
	out    chan []byte
	cancel context.CancelFunc
}

func (s *redisPubSubStream) Messages() <-chan []byte {
	return s.out
}

func (s *redisPubSubStream) Close() error {
	s.cancel()
	return s.pubsub.Close()
}

func NewRedisSubscriber(client *redis.Client) PubSubSubscriber {
	return func(ctx context.Context, channel string) (PubSubStream, error) {
		subCtx, cancel := context.WithCancel(ctx)
		pubsub := client.Subscribe(subCtx, channel)
		if _, err := pubsub.Receive(subCtx); err != nil {
			cancel()
			_ = pubsub.Close()
			return nil, err
		}
		out := make(chan []byte, 64)
		go func() {
			defer close(out)
			for {
				msg, err := pubsub.ReceiveMessage(subCtx)
				if err != nil {
					return
				}
				select {
				case out <- []byte(msg.Payload):
				case <-subCtx.Done():
					return
				}
			}
		}()
		return &redisPubSubStream{pubsub: pubsub, out: out, cancel: cancel}, nil
	}
}

type auctionRoom struct {
	auctionID string
	clients   map[chan []byte]struct{}
	cancel    context.CancelFunc
	stream    PubSubStream
}

// AuctionRoomHub multiplexa 1 única suscripción de Redis hacia N clientes WebSocket locales por subasta.
type AuctionRoomHub struct {
	mu         sync.Mutex
	rooms      map[string]*auctionRoom
	subscriber PubSubSubscriber
}

func NewAuctionRoomHub(subscriber PubSubSubscriber) *AuctionRoomHub {
	return &AuctionRoomHub{
		rooms:      make(map[string]*auctionRoom),
		subscriber: subscriber,
	}
}

func (h *AuctionRoomHub) Join(auctionID string) (<-chan []byte, func(), error) {
	h.mu.Lock()
	room, exists := h.rooms[auctionID]
	if !exists {
		roomCtx, cancel := context.WithCancel(context.Background())
		stream, err := h.subscriber(roomCtx, RedisChannelPrefix()+auctionID)
		if err != nil {
			cancel()
			h.mu.Unlock()
			return nil, nil, err
		}
		room = &auctionRoom{
			auctionID: auctionID,
			clients:   make(map[chan []byte]struct{}),
			cancel:    cancel,
			stream:    stream,
		}
		h.rooms[auctionID] = room
		go h.fanOut(roomCtx, room)
	}

	clientCh := make(chan []byte, 32)
	room.clients[clientCh] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	leave := func() {
		once.Do(func() {
			h.leave(auctionID, clientCh)
		})
	}
	return clientCh, leave, nil
}

func (h *AuctionRoomHub) ActiveRooms() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.rooms)
}

func (h *AuctionRoomHub) fanOut(ctx context.Context, room *auctionRoom) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-room.stream.Messages():
			if !ok {
				return
			}
			h.mu.Lock()
			for clientCh := range room.clients {
				select {
				case clientCh <- payload:
				default:
					// Si el buffer de un cliente lento está lleno, descartamos el mensaje más antiguo
					select {
					case <-clientCh:
					default:
					}
					select {
					case clientCh <- payload:
					default:
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *AuctionRoomHub) leave(auctionID string, clientCh chan []byte) {
	h.mu.Lock()
	room, exists := h.rooms[auctionID]
	if !exists {
		h.mu.Unlock()
		return
	}
	delete(room.clients, clientCh)
	close(clientCh)
	shouldCloseRoom := len(room.clients) == 0
	if shouldCloseRoom {
		delete(h.rooms, auctionID)
	}
	h.mu.Unlock()

	if shouldCloseRoom {
		room.cancel()
		_ = room.stream.Close()
	}
}


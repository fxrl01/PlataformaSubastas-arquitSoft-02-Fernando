package services

import (
	"context"
	"log"
	"time"
)

type AuctionScheduler struct {
	catalog  *AuctionCatalogService
	interval time.Duration
	now      func() time.Time
}

func NewAuctionScheduler(catalog *AuctionCatalogService, interval time.Duration) *AuctionScheduler {
	if interval <= 0 {
		interval = time.Second
	}
	return &AuctionScheduler{catalog: catalog, interval: interval, now: time.Now}
}

func (s *AuctionScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if _, err := s.catalog.CerrarVencidas(ctx, now); err != nil {
				log.Printf("error al cerrar subastas vencidas: %v", err)
			}
		}
	}
}

package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/services"
)

func TestAuctionSchedulerStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scheduler := services.NewAuctionScheduler(services.NuevoAuctionCatalogService(&MockRepo{}), time.Millisecond)
	done := make(chan struct{})
	go func() {
		scheduler.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop after cancellation")
	}
}

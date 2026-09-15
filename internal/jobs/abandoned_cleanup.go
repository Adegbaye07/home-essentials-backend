package jobs

import (
	"context"
	"log/slog"
	"time"

	"homeessentials/backend/internal/repository"
)

const (
	// AbandonedCleanupInterval is how often the worker scans for stale abandoned orders.
	AbandonedCleanupInterval = 3 * time.Minute
	// AbandonedOrderTTL is how long an abandoned order may exist before hard delete.
	AbandonedOrderTTL = 30 * time.Minute
)

const abandonedCleanupTimeout = 30 * time.Second

// AbandonedDeleteCutoff returns the UTC timestamp before which abandoned orders should be deleted.
func AbandonedDeleteCutoff(now time.Time, ttl time.Duration) time.Time {
	return now.UTC().Add(-ttl)
}

// RunAbandonedOrderCleanup runs until ctx is cancelled, deleting abandoned orders past TTL.
func RunAbandonedOrderCleanup(ctx context.Context, orders *repository.OrderRepository, log *slog.Logger) {
	if orders == nil || log == nil {
		return
	}

	runOnce := func() {
		cleanupCtx, cancel := context.WithTimeout(ctx, abandonedCleanupTimeout)
		defer cancel()

		cutoff := AbandonedDeleteCutoff(time.Now(), AbandonedOrderTTL)
		deleted, err := orders.DeleteAbandonedBefore(cleanupCtx, cutoff)
		if err != nil {
			log.Error("abandoned order cleanup failed", "err", err)
			return
		}
		if deleted > 0 {
			log.Info("deleted abandoned orders", "count", deleted, "cutoff", cutoff.Format(time.RFC3339))
		}
	}

	runOnce()

	ticker := time.NewTicker(AbandonedCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

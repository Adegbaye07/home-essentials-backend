package jobs

import (
	"testing"
	"time"
)

func TestAbandonedDeleteCutoff(t *testing.T) {
	now := time.Date(2026, 8, 4, 15, 0, 0, 0, time.UTC)
	cutoff := AbandonedDeleteCutoff(now, AbandonedOrderTTL)
	want := now.Add(-AbandonedOrderTTL)
	if !cutoff.Equal(want) {
		t.Fatalf("cutoff = %v, want %v", cutoff, want)
	}
}

func TestAbandonedCleanupDurations(t *testing.T) {
	if AbandonedCleanupInterval != 3*time.Minute {
		t.Fatalf("interval = %v", AbandonedCleanupInterval)
	}
	if AbandonedOrderTTL != 30*time.Minute {
		t.Fatalf("ttl = %v", AbandonedOrderTTL)
	}
}

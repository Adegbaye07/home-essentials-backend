package controller

import (
	"testing"
	"time"

	"homeessentials/backend/internal/model"
)

func TestMarkOrderPaidIfUnpaid(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		start      model.OrderStatus
		wantChange bool
		wantStatus model.OrderStatus
	}{
		{"pending_payment", model.OrderStatusPendingPayment, true, model.OrderStatusPaid},
		{"abandoned", model.OrderStatusAbandoned, true, model.OrderStatusPaid},
		{"paid noop", model.OrderStatusPaid, false, model.OrderStatusPaid},
		{"packing noop", model.OrderStatusPacking, false, model.OrderStatusPacking},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			abandoned := now.Add(-time.Hour)
			order := &model.Order{
				Status:      tc.start,
				AbandonedAt: &abandoned,
			}
			changed := markOrderPaidIfUnpaid(order, now)
			if changed != tc.wantChange {
				t.Fatalf("changed = %v, want %v", changed, tc.wantChange)
			}
			if order.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", order.Status, tc.wantStatus)
			}
			if tc.wantChange {
				if order.AbandonedAt != nil {
					t.Fatal("expected AbandonedAt cleared on paid transition")
				}
				if order.PaidAt == nil || order.TrackingNumber == "" {
					t.Fatal("expected paidAt and tracking on paid transition")
				}
				if len(order.StatusHistory) != 1 || order.StatusHistory[0].Status != model.OrderStatusPaid {
					t.Fatal("expected paid history entry")
				}
			}
		})
	}
}

func TestAbandonSentinelErrors(t *testing.T) {
	if !IsAbandonEmailMismatch(ErrAbandonEmailMismatch) {
		t.Fatal("IsAbandonEmailMismatch")
	}
	if !IsAbandonConflict(ErrAbandonConflict) {
		t.Fatal("IsAbandonConflict")
	}
}

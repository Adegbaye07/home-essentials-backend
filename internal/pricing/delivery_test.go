package pricing

import (
	"testing"

	"homeessentials/backend/internal/model"
)

func TestDeliveryFeeKobo_pieceOnly(t *testing.T) {
	items := []model.OrderItem{
		{Unit: model.OrderUnitPiece, LineTotalKobo: 1000},
		{Unit: model.OrderUnitPiece, LineTotalKobo: 2000},
	}
	if got := DeliveryFeeKobo(items); got != DeliveryFeePieceOnlyKobo {
		t.Fatalf("got %d want %d", got, DeliveryFeePieceOnlyKobo)
	}
}

func TestDeliveryFeeKobo_bundleFree(t *testing.T) {
	items := []model.OrderItem{
		{Unit: model.OrderUnitPiece, LineTotalKobo: 1000},
		{Unit: model.OrderUnitBundle, LineTotalKobo: 5000},
	}
	if got := DeliveryFeeKobo(items); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestDeliveryFeeKobo_dozenFree(t *testing.T) {
	items := []model.OrderItem{
		{Unit: model.OrderUnitDozen, LineTotalKobo: 2000},
	}
	if got := DeliveryFeeKobo(items); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestDeliveryFeeKobo_empty(t *testing.T) {
	if got := DeliveryFeeKobo(nil); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestItemsSubtotalKobo(t *testing.T) {
	items := []model.OrderItem{
		{LineTotalKobo: 100},
		{LineTotalKobo: 250},
	}
	if got := ItemsSubtotalKobo(items); got != 350 {
		t.Fatalf("got %d want 350", got)
	}
}

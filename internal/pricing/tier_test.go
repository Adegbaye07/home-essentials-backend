package pricing_test

import (
	"errors"
	"strings"
	"testing"

	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pricing"
)

func intPtr(n int) *int { return &n }

func TestUnitPriceForQty(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 1500000},
		{MinQty: 2, MaxQty: intPtr(5), UnitPriceKobo: 1300000},
		{MinQty: 6, UnitPriceKobo: 1000000},
	}

	tests := []struct {
		qty  int
		want int64
	}{
		{1, 1500000},
		{5, 1300000},
		{6, 1000000},
		{100, 1000000},
	}

	for _, tc := range tests {
		got, err := pricing.UnitPriceForQty(tiers, tc.qty)
		if err != nil {
			t.Fatalf("qty %d: %v", tc.qty, err)
		}
		if got != tc.want {
			t.Fatalf("qty %d: got %d want %d", tc.qty, got, tc.want)
		}
	}
}

func TestValidateTiers_overlap(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(5), UnitPriceKobo: 100},
		{MinQty: 3, MaxQty: intPtr(10), UnitPriceKobo: 90},
	}
	err := pricing.ValidateTiers(model.SizeM, tiers)
	if !errors.Is(err, pricing.ErrOverlappingTier) {
		t.Fatalf("want overlap error, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "second price tier for size M") {
		t.Fatalf("expected readable overlap message, got %q", msg)
	}
	if !strings.Contains(msg, "overlaps the tier above") {
		t.Fatalf("expected overlap detail, got %q", msg)
	}
}

func TestValidateTiers_gap(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 100},
		{MinQty: 3, UnitPriceKobo: 90},
	}
	err := pricing.ValidateTiers(model.SizeM, tiers)
	if !errors.Is(err, pricing.ErrGapInTiers) {
		t.Fatalf("want gap error, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "second price tier for size M") {
		t.Fatalf("expected readable gap message, got %q", msg)
	}
	if !strings.Contains(msg, "must start at quantity 2") {
		t.Fatalf("expected expected min in message, got %q", msg)
	}
	if !strings.Contains(msg, "but it starts at 3") {
		t.Fatalf("expected actual min in message, got %q", msg)
	}
	if strings.Contains(msg, "gap in quantity tier coverage") {
		t.Fatalf("expected clean user message without sentinel text, got %q", msg)
	}
}

func TestValidateTiers_firstTierMustStartAtOne(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 2, MaxQty: intPtr(5), UnitPriceKobo: 100},
	}
	err := pricing.ValidateTiers(model.SizeL, tiers)
	if !errors.Is(err, pricing.ErrGapInTiers) {
		t.Fatalf("want gap error, got %v", err)
	}
	if !strings.Contains(err.Error(), "first price tier for size L must start at quantity 1") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestValidateTiers_openEndedNotLast(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, UnitPriceKobo: 100},
		{MinQty: 6, MaxQty: intPtr(10), UnitPriceKobo: 90},
	}
	err := pricing.ValidateTiers(model.SizeM, tiers)
	if !errors.Is(err, pricing.ErrOverlappingTier) {
		t.Fatalf("want overlap error, got %v", err)
	}
	if !strings.Contains(err.Error(), "first price tier for size M") {
		t.Fatalf("unexpected message: %v", err)
	}
	if !strings.Contains(err.Error(), "is open-ended, but another tier follows it") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestValidateTiers_cappedLastTier_valid(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 1500000, DeliveryDays: 14},
		{MinQty: 2, MaxQty: intPtr(5), UnitPriceKobo: 1300000, DeliveryDays: 10},
		{MinQty: 6, MaxQty: intPtr(10), UnitPriceKobo: 1000000, DeliveryDays: 7},
	}
	if err := pricing.ValidateTiers(model.SizeM, tiers); err != nil {
		t.Fatalf("expected capped last tier to validate, got %v", err)
	}
}

func TestValidateTiers_openEndedLastTier_stillValid(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(5), UnitPriceKobo: 1300000, DeliveryDays: 10},
		{MinQty: 6, UnitPriceKobo: 1000000, DeliveryDays: 7},
	}
	if err := pricing.ValidateTiers(model.SizeM, tiers); err != nil {
		t.Fatalf("expected open-ended last tier to validate, got %v", err)
	}
}

func TestValidateTiers_empty(t *testing.T) {
	err := pricing.ValidateTiers(model.SizeM, nil)
	if !errors.Is(err, pricing.ErrInvalidTiers) {
		t.Fatalf("want invalid tiers error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Size M must have at least one price tier") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestUnitPriceForQty_cappedLastTier(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 1500000},
		{MinQty: 2, MaxQty: intPtr(5), UnitPriceKobo: 1300000},
		{MinQty: 6, MaxQty: intPtr(10), UnitPriceKobo: 1000000},
	}

	got, err := pricing.UnitPriceForQty(tiers, 10)
	if err != nil {
		t.Fatalf("qty 10: %v", err)
	}
	if got != 1000000 {
		t.Fatalf("qty 10: got %d want 1000000", got)
	}

	_, err = pricing.UnitPriceForQty(tiers, 11)
	if !errors.Is(err, pricing.ErrNoTierForQty) {
		t.Fatalf("qty 11: want ErrNoTierForQty, got %v", err)
	}
}

func TestValidateSizeVariants_cappedLastTier(t *testing.T) {
	sizes := []model.SizeVariant{
		{
			Code: model.SizeM,
			Tiers: []model.QtyTier{
				{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 100000, DeliveryDays: 14},
				{MinQty: 2, MaxQty: intPtr(10), UnitPriceKobo: 80000, DeliveryDays: 7},
			},
		},
	}
	if err := pricing.ValidateSizeVariants(sizes); err != nil {
		t.Fatalf("expected valid size variants, got %v", err)
	}
}

func TestValidateSizeVariants_noDoubleSizePrefix(t *testing.T) {
	sizes := []model.SizeVariant{
		{
			Code: model.SizeM,
			Tiers: []model.QtyTier{
				{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 100},
				{MinQty: 3, UnitPriceKobo: 90},
			},
		},
	}
	err := pricing.ValidateSizeVariants(sizes)
	if err == nil {
		t.Fatal("expected gap error")
	}
	if strings.Count(err.Error(), "size M:") > 0 {
		t.Fatalf("expected no legacy size prefix, got %q", err.Error())
	}
}

func TestValidateTierDeliveryDays_readableMessage(t *testing.T) {
	tiers := []model.QtyTier{
		{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 100, DeliveryDays: 14},
		{MinQty: 2, UnitPriceKobo: 90, DeliveryDays: 0},
	}
	err := pricing.ValidateTierDeliveryDays(model.SizeM, tiers)
	if err == nil {
		t.Fatal("expected delivery days error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "second price tier for size M") {
		t.Fatalf("unexpected message: %q", msg)
	}
	if !strings.Contains(msg, "delivery days of at least 1") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestTierOrdinal(t *testing.T) {
	tests := []struct {
		i, total int
		want     string
	}{
		{0, 1, "first"},
		{0, 3, "first"},
		{1, 3, "second"},
		{2, 3, "last"},
		{1, 2, "second"},
	}
	for _, tc := range tests {
		if got := pricing.TierOrdinal(tc.i, tc.total); got != tc.want {
			t.Fatalf("TierOrdinal(%d, %d) = %q, want %q", tc.i, tc.total, got, tc.want)
		}
	}
}

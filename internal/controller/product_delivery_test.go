package controller

import (
	"strings"
	"testing"

	"homeessentials/backend/internal/model"
)

func validProductInputForDeliveryTest() ProductInput {
	return ProductInput{
		Title:       "Test",
		Description: "Desc",
		Category:    model.CategoryTote,
		Colors:      []string{"black"},
		ColorImages: []model.ColorImage{{Color: "black", ImageURL: "https://example.com/x.jpg"}},
		Active:      true,
		Sizes: []model.SizeVariant{
			{
				Code: model.SizeM,
				Tiers: []model.QtyTier{
					{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 100000, DeliveryDays: 14},
					{MinQty: 2, UnitPriceKobo: 80000, DeliveryDays: 7},
				},
			},
		},
	}
}

func TestValidateProductInput_deliveryDaysRequired(t *testing.T) {
	in := validProductInputForDeliveryTest()
	in.Sizes[0].Tiers[0].DeliveryDays = 0

	err := validateProductInput(in)
	if err == nil {
		t.Fatal("expected error for missing deliveryDays")
	}
	if !strings.Contains(err.Error(), "first price tier for size M") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "delivery days of at least 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProductInput_deliveryDaysValid(t *testing.T) {
	err := validateProductInput(validProductInputForDeliveryTest())
	if err != nil {
		t.Fatalf("expected valid input: %v", err)
	}
}

func TestValidateProductInput_deliveryDaysPerTier(t *testing.T) {
	in := validProductInputForDeliveryTest()
	in.Sizes[0].Tiers[1].DeliveryDays = -1

	err := validateProductInput(in)
	if err == nil {
		t.Fatal("expected error for negative deliveryDays")
	}
	if !strings.Contains(err.Error(), "second price tier for size M") {
		t.Fatalf("expected tier index in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "delivery days of at least 1") {
		t.Fatalf("expected delivery days message, got: %v", err)
	}
}

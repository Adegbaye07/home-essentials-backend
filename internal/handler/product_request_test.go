package handler

import (
	"strings"
	"testing"
)

func TestRequestToInput_mapsDeliveryDays(t *testing.T) {
	max1 := 1
	in, err := requestToInput(productRequest{
		Title:       "T",
		Description: "D",
		Category:    "tote",
		Colors:      []string{"black"},
		ColorImages: []colorImageDTO{{Color: "black", ImageURL: "https://example.com/x.jpg"}},
		Active:      true,
		Sizes: []productSizeDTO{
			{
				Code: "M",
				Tiers: []qtyTierDTO{
					{MinQty: 1, MaxQty: &max1, UnitPriceKobo: 100000, DeliveryDays: 21},
					{MinQty: 2, UnitPriceKobo: 90000, DeliveryDays: 14},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Sizes) != 1 || len(in.Sizes[0].Tiers) != 2 {
		t.Fatalf("sizes: %+v", in.Sizes)
	}
	if in.Sizes[0].Tiers[0].DeliveryDays != 21 || in.Sizes[0].Tiers[1].DeliveryDays != 14 {
		t.Fatalf("delivery days not mapped: %+v", in.Sizes[0].Tiers)
	}
}

func TestRequestToInput_zeroDeliveryDaysPreservedForValidation(t *testing.T) {
	in, err := requestToInput(productRequest{
		Title:       "T",
		Description: "D",
		Category:    "tote",
		Colors:      []string{"black"},
		ColorImages: []colorImageDTO{{Color: "black", ImageURL: "https://example.com/x.jpg"}},
		Active:      true,
		Sizes: []productSizeDTO{
			{
				Code: "S",
				Tiers: []qtyTierDTO{
					{MinQty: 1, UnitPriceKobo: 100000, DeliveryDays: 0},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if in.Sizes[0].Tiers[0].DeliveryDays != 0 {
		t.Fatal("expected zero delivery days from JSON omit/default")
	}
}

func TestRequestToInput_invalidSizeStillFails(t *testing.T) {
	_, err := requestToInput(productRequest{
		Category: "tote",
		Sizes: []productSizeDTO{
			{Code: "XS", Tiers: []qtyTierDTO{{MinQty: 1, UnitPriceKobo: 100, DeliveryDays: 7}}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("expected invalid size error, got %v", err)
	}
}

// Ensure category parse still works (regression).
func TestRequestToInput_category(t *testing.T) {
	_, err := requestToInput(productRequest{Category: "not-a-cat"})
	if err == nil {
		t.Fatal("expected category error")
	}
}

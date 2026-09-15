package handler

import (
	"strings"
	"testing"
)

func TestRequestToInput_mapsRugPricing(t *testing.T) {
	in, err := requestToInput(productRequest{
		Title:       "Runner rug",
		Description: "Soft runner",
		Category:    "rugs",
		Variants:    []string{"beige"},
		VariantImages: []variantImageDTO{
			{Variant: "beige", ImageURL: "https://example.com/beige.jpg"},
		},
		Active: true,
		SizePricings: []sizePricingDTO{
			{Size: "2 x 5 ft", PiecePriceKobo: 500000, BundlePriceKobo: 4500000, PiecesPerBundle: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.SizePricings) != 1 {
		t.Fatalf("sizePricings: %+v", in.SizePricings)
	}
	if in.SizePricings[0].PiecesPerBundle != 10 {
		t.Fatalf("piecesPerBundle: %+v", in.SizePricings[0])
	}
	if in.CleaningPricing != nil {
		t.Fatal("expected nil cleaningPricing")
	}
}

func TestRequestToInput_mapsCleaningPricing(t *testing.T) {
	in, err := requestToInput(productRequest{
		Title:       "Mop",
		Description: "Floor mop",
		Category:    "cleaning_essentials",
		Variants:    []string{"standard"},
		VariantImages: []variantImageDTO{
			{Variant: "standard", ImageURL: "https://example.com/mop.jpg"},
		},
		Active: true,
		CleaningPricing: &cleaningPricingDTO{
			PiecePriceKobo: 200000,
			DozenPriceKobo: 2000000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if in.CleaningPricing == nil || in.CleaningPricing.DozenPriceKobo != 2000000 {
		t.Fatalf("cleaningPricing: %+v", in.CleaningPricing)
	}
}

func TestRequestToInput_invalidCategory(t *testing.T) {
	_, err := requestToInput(productRequest{Category: "tote"})
	if err == nil || !strings.Contains(err.Error(), "invalid category") {
		t.Fatalf("expected category error, got %v", err)
	}
}

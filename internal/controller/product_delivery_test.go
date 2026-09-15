package controller

import (
	"strings"
	"testing"

	"homeessentials/backend/internal/model"
)

func validRugProductInput() ProductInput {
	return ProductInput{
		Title:       "Test rug",
		Description: "Desc",
		Category:    model.CategoryRugs,
		Variants:    []string{"beige"},
		VariantImages: []model.VariantImage{
			{Variant: "beige", ImageURL: "https://example.com/x.jpg"},
		},
		Active: true,
		SizePricings: []model.SizePricing{
			{Size: "2 x 5 ft", PiecePriceKobo: 500000, BundlePriceKobo: 4500000, PiecesPerBundle: 10},
		},
	}
}

func TestValidateProductInput_rugOK(t *testing.T) {
	_, err := normalizeAndValidateProductInput(validRugProductInput())
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateProductInput_cleaningOK(t *testing.T) {
	in := ProductInput{
		Title:       "Mop",
		Description: "Desc",
		Category:    model.CategoryCleaningEssentials,
		Variants:    []string{"standard"},
		VariantImages: []model.VariantImage{
			{Variant: "standard", ImageURL: "https://example.com/mop.jpg"},
		},
		Active: true,
		CleaningPricing: &model.CleaningPricing{
			PiecePriceKobo: 100000,
			DozenPriceKobo: 1000000,
		},
	}
	got, err := normalizeAndValidateProductInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.CleaningPricing == nil || len(got.SizePricings) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateProductInput_requiresVariantImage(t *testing.T) {
	in := validRugProductInput()
	in.VariantImages = nil
	_, err := normalizeAndValidateProductInput(in)
	if err == nil || !strings.Contains(err.Error(), "image required") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateProductInput_cleaningRejectsSizes(t *testing.T) {
	in := ProductInput{
		Title:       "Mop",
		Description: "Desc",
		Category:    model.CategoryCleaningEssentials,
		Variants:    []string{"standard"},
		VariantImages: []model.VariantImage{
			{Variant: "standard", ImageURL: "https://example.com/mop.jpg"},
		},
		Active: true,
		SizePricings: []model.SizePricing{
			{Size: "x", PiecePriceKobo: 1, BundlePriceKobo: 1, PiecesPerBundle: 1},
		},
		CleaningPricing: &model.CleaningPricing{PiecePriceKobo: 1, DozenPriceKobo: 1},
	}
	_, err := normalizeAndValidateProductInput(in)
	if err == nil || !strings.Contains(err.Error(), "must not have sizePricings") {
		t.Fatalf("got %v", err)
	}
}

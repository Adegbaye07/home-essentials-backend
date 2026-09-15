package pricing

import (
	"testing"

	"homeessentials/backend/internal/model"
)

func TestValidateProductPricing_mats(t *testing.T) {
	err := ValidateProductPricing(model.CategoryRugs, []model.SizePricing{
		{Size: "2 x 5 ft", PiecePriceKobo: 500000, BundlePriceKobo: 4500000, PiecesPerBundle: 10},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateProductPricing_cleaning(t *testing.T) {
	err := ValidateProductPricing(model.CategoryCleaningEssentials, nil, &model.CleaningPricing{
		PiecePriceKobo: 200000,
		DozenPriceKobo: 2000000,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateProductPricing_cleaningRejectsSizes(t *testing.T) {
	err := ValidateProductPricing(model.CategoryCleaningEssentials, []model.SizePricing{
		{Size: "x", PiecePriceKobo: 1, BundlePriceKobo: 1, PiecesPerBundle: 1},
	}, &model.CleaningPricing{PiecePriceKobo: 1, DozenPriceKobo: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateProductPricing_matsRejectCleaning(t *testing.T) {
	err := ValidateProductPricing(model.CategoryFootMats, []model.SizePricing{
		{Size: "60cm", PiecePriceKobo: 1, BundlePriceKobo: 1, PiecesPerBundle: 2},
	}, &model.CleaningPricing{PiecePriceKobo: 1, DozenPriceKobo: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveLinePrice_bundle(t *testing.T) {
	p := model.Product{
		Category: model.CategoryDoorMats,
		SizePricings: []model.SizePricing{
			{Size: "2ft x 3ft", PiecePriceKobo: 1000, BundlePriceKobo: 9000, PiecesPerBundle: 10},
		},
	}
	got, err := ResolveLinePrice(p, "2ft x 3ft", model.OrderUnitBundle)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPriceKobo != 9000 || got.PiecesPerBundle != 10 {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveLinePrice_dozen(t *testing.T) {
	p := model.Product{
		Category: model.CategoryCleaningEssentials,
		CleaningPricing: &model.CleaningPricing{
			PiecePriceKobo: 500,
			DozenPriceKobo: 5000,
		},
	}
	got, err := ResolveLinePrice(p, "", model.OrderUnitDozen)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPriceKobo != 5000 {
		t.Fatalf("got %+v", got)
	}
}

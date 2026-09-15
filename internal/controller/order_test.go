package controller

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/model"
)

func TestBuildOrderItems_pieceAndBundle(t *testing.T) {
	pid := primitive.NewObjectID()
	products := map[primitive.ObjectID]model.Product{
		pid: {
			ID:       pid,
			Title:    "Door mat",
			Category: model.CategoryDoorMats,
			Active:   true,
			Variants: []string{"brown"},
			VariantImages: []model.VariantImage{
				{Variant: "brown", ImageURL: "https://example.com/brown.jpg"},
			},
			SizePricings: []model.SizePricing{
				{Size: "2 x 3 ft", PiecePriceKobo: 1500000, BundlePriceKobo: 12000000, PiecesPerBundle: 10},
			},
		},
	}

	lines := []OrderLineInput{
		{ProductID: pid, Variant: "brown", Size: "2 x 3 ft", Unit: model.OrderUnitPiece, Quantity: 2},
		{ProductID: pid, Variant: "brown", Size: "2 x 3 ft", Unit: model.OrderUnitBundle, Quantity: 1},
	}

	items, total, err := buildOrderItems(products, lines)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d", len(items))
	}
	want := int64(2*1500000 + 12000000)
	if total != want {
		t.Fatalf("total = %d want %d", total, want)
	}
	if items[0].ImageURL != "https://example.com/brown.jpg" {
		t.Fatalf("image = %q", items[0].ImageURL)
	}
	if items[1].PiecesPerBundle != 10 {
		t.Fatalf("piecesPerBundle = %d", items[1].PiecesPerBundle)
	}
}

func TestBuildOrderItems_cleaningDozen(t *testing.T) {
	pid := primitive.NewObjectID()
	products := map[primitive.ObjectID]model.Product{
		pid: {
			ID:       pid,
			Title:    "Brush",
			Category: model.CategoryCleaningEssentials,
			Active:   true,
			Variants: []string{"blue"},
			VariantImages: []model.VariantImage{
				{Variant: "blue", ImageURL: "https://example.com/blue.jpg"},
			},
			CleaningPricing: &model.CleaningPricing{
				PiecePriceKobo: 200000,
				DozenPriceKobo: 2000000,
			},
		},
	}

	items, total, err := buildOrderItems(products, []OrderLineInput{
		{ProductID: pid, Variant: "blue", Unit: model.OrderUnitDozen, Quantity: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2000000 || items[0].Unit != model.OrderUnitDozen {
		t.Fatalf("got total=%d unit=%s", total, items[0].Unit)
	}
}

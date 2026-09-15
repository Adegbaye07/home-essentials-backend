package controller

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/model"
)

func intPtr(n int) *int { return &n }

func TestBuildOrderItems_mixedSizesAndTiers(t *testing.T) {
	pid := primitive.NewObjectID()
	products := map[primitive.ObjectID]model.Product{
		pid: {
			ID:     pid,
			Title:  "Test Bag",
			Active: true,
			Colors: []string{"black"},
			ColorImages: []model.ColorImage{
				{Color: "black", ImageURL: "https://example.com/black.jpg"},
			},
			Sizes: []model.SizeVariant{
				{
					Code: model.SizeM,
					Tiers: []model.QtyTier{
						{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 1500000},
						{MinQty: 2, MaxQty: intPtr(5), UnitPriceKobo: 1300000},
						{MinQty: 6, UnitPriceKobo: 1000000},
					},
				},
				{
					Code: model.SizeL,
					Tiers: []model.QtyTier{
						{MinQty: 1, MaxQty: intPtr(1), UnitPriceKobo: 1600000},
						{MinQty: 2, UnitPriceKobo: 1200000},
					},
				},
			},
		},
	}

	lines := []OrderLineInput{
		{ProductID: pid, Size: model.SizeM, Color: "black", Quantity: 1},
		{ProductID: pid, Size: model.SizeM, Color: "black", Quantity: 5},
		{ProductID: pid, Size: model.SizeL, Color: "black", Quantity: 2},
	}

	items, total, err := buildOrderItems(products, lines)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("items len = %d, want 3", len(items))
	}

	want := int64(1500000 + 5*1300000 + 2*1200000)
	if total != want {
		t.Fatalf("total = %d, want %d", total, want)
	}

	if items[0].LineTotalKobo != 1500000 {
		t.Fatalf("line 1 total = %d", items[0].LineTotalKobo)
	}
	if items[0].ImageURL != "https://example.com/black.jpg" {
		t.Fatalf("line 1 imageUrl = %q", items[0].ImageURL)
	}
	if items[1].UnitPriceKobo != 1300000 || items[1].LineTotalKobo != 5*1300000 {
		t.Fatalf("line 2 pricing wrong: unit=%d line=%d", items[1].UnitPriceKobo, items[1].LineTotalKobo)
	}
}

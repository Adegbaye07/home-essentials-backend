package model

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VariantImage struct {
	Variant  string `bson:"variant" json:"variant"`
	ImageURL string `bson:"imageUrl" json:"imageUrl"`
}

// SizePricing is piece/bundle pricing for one free-text size (mats and rugs).
type SizePricing struct {
	Size            string `bson:"size" json:"size"`
	PiecePriceKobo  int64  `bson:"piecePriceKobo" json:"piecePriceKobo"`
	BundlePriceKobo int64  `bson:"bundlePriceKobo" json:"bundlePriceKobo"`
	PiecesPerBundle int    `bson:"piecesPerBundle" json:"piecesPerBundle"`
}

// CleaningPricing is piece/dozen pricing for cleaning essentials (no sizes).
type CleaningPricing struct {
	PiecePriceKobo int64 `bson:"piecePriceKobo" json:"piecePriceKobo"`
	DozenPriceKobo int64 `bson:"dozenPriceKobo" json:"dozenPriceKobo"`
}

type Product struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	Description     string             `bson:"description" json:"description"`
	Category        Category           `bson:"category" json:"category"`
	Variants        []string           `bson:"variants" json:"variants"`
	VariantImages   []VariantImage     `bson:"variantImages" json:"variantImages"`
	SizePricings    []SizePricing      `bson:"sizePricings,omitempty" json:"sizePricings,omitempty"`
	CleaningPricing *CleaningPricing   `bson:"cleaningPricing,omitempty" json:"cleaningPricing,omitempty"`
	Active          bool               `bson:"active" json:"active"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ImageURLForVariant returns the image URL for a variant, or empty if unknown.
func (p Product) ImageURLForVariant(variant string) string {
	for _, vi := range p.VariantImages {
		if equalFoldTrim(vi.Variant, variant) {
			return vi.ImageURL
		}
	}
	return ""
}

// HasVariant reports whether the product lists the given variant.
func (p Product) HasVariant(variant string) bool {
	for _, v := range p.Variants {
		if equalFoldTrim(v, variant) {
			return true
		}
	}
	return false
}

// SizePricingFor returns pricing for a size label, or nil.
func (p Product) SizePricingFor(size string) *SizePricing {
	for i := range p.SizePricings {
		if equalFoldTrim(p.SizePricings[i].Size, size) {
			return &p.SizePricings[i]
		}
	}
	return nil
}

func equalFoldTrim(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

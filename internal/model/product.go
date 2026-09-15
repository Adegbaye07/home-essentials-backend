package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QtyTier struct {
	MinQty        int   `bson:"minQty" json:"minQty"`
	MaxQty        *int  `bson:"maxQty,omitempty" json:"maxQty,omitempty"`
	UnitPriceKobo int64 `bson:"unitPriceKobo" json:"unitPriceKobo"`
	DeliveryDays  int   `bson:"deliveryDays" json:"deliveryDays"`
}

type SizeVariant struct {
	Code  SizeCode  `bson:"code" json:"code"`
	Tiers []QtyTier `bson:"tiers" json:"tiers"`
}

type ColorImage struct {
	Color    string `bson:"color" json:"color"`
	ImageURL string `bson:"imageUrl" json:"imageUrl"`
}

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Category    Category           `bson:"category" json:"category"`
	Colors      []string           `bson:"colors" json:"colors"`
	ColorImages []ColorImage       `bson:"colorImages" json:"colorImages"`
	Active      bool               `bson:"active" json:"active"`
	Sizes       []SizeVariant      `bson:"sizes" json:"sizes"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ImageURLForColor returns the product image URL for the given color, or empty if unknown.
func (p Product) ImageURLForColor(color string) string {
	for _, ci := range p.ColorImages {
		if ci.Color == color {
			return ci.ImageURL
		}
	}
	return ""
}

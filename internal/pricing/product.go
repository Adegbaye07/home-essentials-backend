package pricing

import (
	"errors"
	"fmt"
	"strings"

	"homeessentials/backend/internal/model"
)

var (
	ErrInvalidPricing = errors.New("invalid product pricing")
	ErrNoPriceForUnit = errors.New("no price for unit")
)

// ValidateProductPricing validates size/cleaning pricing for a category.
func ValidateProductPricing(category model.Category, sizes []model.SizePricing, cleaning *model.CleaningPricing) error {
	if category.IsCleaning() {
		if len(sizes) > 0 {
			return fmt.Errorf("cleaning essentials must not have sizePricings: %w", ErrInvalidPricing)
		}
		if cleaning == nil {
			return fmt.Errorf("cleaningPricing is required for cleaning essentials: %w", ErrInvalidPricing)
		}
		if cleaning.PiecePriceKobo < 1 {
			return fmt.Errorf("cleaning piecePriceKobo must be at least 1: %w", ErrInvalidPricing)
		}
		if cleaning.DozenPriceKobo < 1 {
			return fmt.Errorf("cleaning dozenPriceKobo must be at least 1: %w", ErrInvalidPricing)
		}
		return nil
	}

	if cleaning != nil {
		return fmt.Errorf("cleaningPricing is only allowed for cleaning essentials: %w", ErrInvalidPricing)
	}
	if len(sizes) == 0 {
		return fmt.Errorf("at least one sizePricing is required: %w", ErrInvalidPricing)
	}

	seen := make(map[string]struct{}, len(sizes))
	for i, sp := range sizes {
		size := strings.TrimSpace(sp.Size)
		if size == "" {
			return fmt.Errorf("sizePricings[%d]: size is required: %w", i, ErrInvalidPricing)
		}
		key := strings.ToLower(size)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate size %q: %w", size, ErrInvalidPricing)
		}
		seen[key] = struct{}{}
		if sp.PiecePriceKobo < 1 {
			return fmt.Errorf("size %q: piecePriceKobo must be at least 1: %w", size, ErrInvalidPricing)
		}
		if sp.BundlePriceKobo < 1 {
			return fmt.Errorf("size %q: bundlePriceKobo must be at least 1: %w", size, ErrInvalidPricing)
		}
		if sp.PiecesPerBundle < 1 {
			return fmt.Errorf("size %q: piecesPerBundle must be at least 1: %w", size, ErrInvalidPricing)
		}
	}
	return nil
}

// LinePriceResult is the resolved unit price for an order line.
type LinePriceResult struct {
	UnitPriceKobo   int64
	PiecesPerBundle int // set when unit is bundle
}

// ResolveLinePrice returns the unit price for a product line (server-side).
func ResolveLinePrice(p model.Product, size string, unit model.OrderUnit) (LinePriceResult, error) {
	if p.Category.IsCleaning() {
		if strings.TrimSpace(size) != "" {
			return LinePriceResult{}, fmt.Errorf("size is not allowed for cleaning essentials: %w", ErrInvalidPricing)
		}
		if p.CleaningPricing == nil {
			return LinePriceResult{}, fmt.Errorf("product has no cleaning pricing: %w", ErrInvalidPricing)
		}
		switch unit {
		case model.OrderUnitPiece:
			return LinePriceResult{UnitPriceKobo: p.CleaningPricing.PiecePriceKobo}, nil
		case model.OrderUnitDozen:
			return LinePriceResult{UnitPriceKobo: p.CleaningPricing.DozenPriceKobo}, nil
		case model.OrderUnitBundle:
			return LinePriceResult{}, fmt.Errorf("bundle is not available for cleaning essentials: %w", ErrNoPriceForUnit)
		default:
			return LinePriceResult{}, fmt.Errorf("invalid unit %q: %w", unit, ErrNoPriceForUnit)
		}
	}

	if unit == model.OrderUnitDozen {
		return LinePriceResult{}, fmt.Errorf("dozen is only available for cleaning essentials: %w", ErrNoPriceForUnit)
	}
	sp := p.SizePricingFor(size)
	if sp == nil {
		return LinePriceResult{}, fmt.Errorf("size %q is not available for this product: %w", size, ErrNoPriceForUnit)
	}
	switch unit {
	case model.OrderUnitPiece:
		return LinePriceResult{UnitPriceKobo: sp.PiecePriceKobo}, nil
	case model.OrderUnitBundle:
		return LinePriceResult{UnitPriceKobo: sp.BundlePriceKobo, PiecesPerBundle: sp.PiecesPerBundle}, nil
	default:
		return LinePriceResult{}, fmt.Errorf("invalid unit %q: %w", unit, ErrNoPriceForUnit)
	}
}

package pricing

import (
	"errors"
	"fmt"
	"sort"

	"homeessentials/backend/internal/model"
)

var (
	ErrNoTierForQty    = errors.New("no price tier for quantity")
	ErrInvalidTiers    = errors.New("invalid quantity tiers")
	ErrOverlappingTier = errors.New("overlapping quantity tiers")
	ErrGapInTiers      = errors.New("gap in quantity tier coverage")
)

// UnitPriceForQty returns the unit price in kobo for qty using the tier table.
func UnitPriceForQty(tiers []model.QtyTier, qty int) (int64, error) {
	if qty < 1 {
		return 0, fmt.Errorf("quantity must be at least 1")
	}

	for _, t := range tiers {
		if qty < t.MinQty {
			continue
		}
		if t.MaxQty != nil && qty > *t.MaxQty {
			continue
		}
		return t.UnitPriceKobo, nil
	}

	return 0, ErrNoTierForQty
}

func sortedTiersCopy(tiers []model.QtyTier) []model.QtyTier {
	sorted := make([]model.QtyTier, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MinQty < sorted[j].MinQty
	})
	return sorted
}

// ValidateTiers checks tier rows for a single size variant.
func ValidateTiers(size model.SizeCode, tiers []model.QtyTier) error {
	if len(tiers) == 0 {
		return sizeValidationError(size, "must have at least one price tier.", ErrInvalidTiers)
	}

	sorted := sortedTiersCopy(tiers)
	total := len(sorted)

	if sorted[0].MinQty != 1 {
		return tierValidationError(size, 0, total, "must start at quantity 1", ErrGapInTiers)
	}

	openEnded := 0
	for i, t := range sorted {
		if t.MinQty < 1 {
			return tierValidationError(size, i, total, "must have a minimum quantity of at least 1", ErrInvalidTiers)
		}
		if t.UnitPriceKobo < 1 {
			return tierValidationError(size, i, total, "must have a price greater than zero", ErrInvalidTiers)
		}
		if t.MaxQty != nil {
			if *t.MaxQty < t.MinQty {
				return tierValidationError(
					size,
					i,
					total,
					fmt.Sprintf("has a maximum (%d) less than its minimum (%d)", *t.MaxQty, t.MinQty),
					ErrInvalidTiers,
				)
			}
		} else {
			openEnded++
		}

		if i > 0 {
			prev := sorted[i-1]
			if prev.MaxQty == nil {
				return tierValidationError(
					size,
					i-1,
					total,
					"is open-ended, but another tier follows it; only the last tier may be open-ended",
					ErrOverlappingTier,
				)
			}
			if t.MinQty <= *prev.MaxQty {
				return tierValidationError(size, i, total, "overlaps the tier above", ErrOverlappingTier)
			}
			expectedMin := *prev.MaxQty + 1
			if t.MinQty != expectedMin {
				return tierValidationError(
					size,
					i,
					total,
					fmt.Sprintf(
						"must start at quantity %d (one more than the maximum of %d on the tier above), but it starts at %d",
						expectedMin,
						*prev.MaxQty,
						t.MinQty,
					),
					ErrGapInTiers,
				)
			}
		}
	}

	if openEnded > 1 {
		return sizeValidationError(
			size,
			"has more than one open-ended price tier; only the last tier may be open-ended.",
			ErrInvalidTiers,
		)
	}

	return nil
}

// ValidateTierDeliveryDays checks deliveryDays on each tier row for one size.
func ValidateTierDeliveryDays(size model.SizeCode, tiers []model.QtyTier) error {
	sorted := sortedTiersCopy(tiers)
	total := len(sorted)
	for i, t := range sorted {
		if t.DeliveryDays < 1 {
			return fmt.Errorf("%s", TierDeliveryDaysError(size, i, total))
		}
	}
	return nil
}

// ValidateSizeVariants validates tiers per size and unique size codes.
func ValidateSizeVariants(sizes []model.SizeVariant) error {
	if len(sizes) == 0 {
		return fmt.Errorf("at least one size is required")
	}

	seen := make(map[model.SizeCode]struct{}, len(sizes))
	for _, sv := range sizes {
		if !sv.Code.Valid() {
			return fmt.Errorf("invalid size code %q", sv.Code)
		}
		if _, ok := seen[sv.Code]; ok {
			return fmt.Errorf("duplicate size %q", sv.Code)
		}
		seen[sv.Code] = struct{}{}
		if err := ValidateTiers(sv.Code, sv.Tiers); err != nil {
			return err
		}
	}

	return nil
}

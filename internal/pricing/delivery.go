package pricing

import "homeessentials/backend/internal/model"

// DeliveryFeePieceOnlyKobo is charged once per order when every line is piece.
const DeliveryFeePieceOnlyKobo int64 = 200_000 // ₦2,000

// DeliveryFeeKobo returns the flat delivery fee for an order.
// Free when any line is bundle or dozen; otherwise ₦2,000 for piece-only carts.
func DeliveryFeeKobo(items []model.OrderItem) int64 {
	if len(items) == 0 {
		return 0
	}
	for _, it := range items {
		switch it.Unit {
		case model.OrderUnitBundle, model.OrderUnitDozen:
			return 0
		}
	}
	return DeliveryFeePieceOnlyKobo
}

// ItemsSubtotalKobo sums line totals (excludes delivery).
func ItemsSubtotalKobo(items []model.OrderItem) int64 {
	var sum int64
	for _, it := range items {
		sum += it.LineTotalKobo
	}
	return sum
}

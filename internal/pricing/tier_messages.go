package pricing

import (
	"fmt"

	"homeessentials/backend/internal/model"
)

// TierOrdinal returns a user-facing label for a tier position (0-based index).
func TierOrdinal(i, total int) string {
	if total >= 3 && i == total-1 {
		return "last"
	}

	ordinals := []string{
		"first", "second", "third", "fourth", "fifth",
		"sixth", "seventh", "eighth", "ninth", "tenth",
	}
	if i >= 0 && i < len(ordinals) {
		return ordinals[i]
	}

	return fmt.Sprintf("%dth", i+1)
}

func tierMessage(size model.SizeCode, i, total int, detail string) string {
	return fmt.Sprintf("The %s price tier for size %s %s", TierOrdinal(i, total), size, detail)
}

type validationError struct {
	message string
	cause   error
}

func (e *validationError) Error() string { return e.message }
func (e *validationError) Unwrap() error { return e.cause }

func tierValidationError(size model.SizeCode, i, total int, detail string, sentinel error) error {
	return &validationError{message: tierMessage(size, i, total, detail), cause: sentinel}
}

func sizeValidationError(size model.SizeCode, message string, sentinel error) error {
	return &validationError{message: fmt.Sprintf("Size %s %s", size, message), cause: sentinel}
}

// TierDeliveryDaysError describes invalid deliveryDays on a tier row.
func TierDeliveryDaysError(size model.SizeCode, tierIndex, tierCount int) string {
	return fmt.Sprintf(
		"The %s price tier for size %s must have delivery days of at least 1.",
		TierOrdinal(tierIndex, tierCount),
		size,
	)
}

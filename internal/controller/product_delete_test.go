package controller

import (
	"errors"
	"testing"
)

func TestProductDeleteSentinelErrors(t *testing.T) {
	if ErrProductDeletePendingOrder.Error() != "There is a pending order for this item." {
		t.Fatalf("pending message: %q", ErrProductDeletePendingOrder.Error())
	}
	if ErrProductDeleteIncompleteOrder.Error() != "There is an incomplete order for this item." {
		t.Fatalf("incomplete message: %q", ErrProductDeleteIncompleteOrder.Error())
	}
	if errors.Is(ErrProductDeletePendingOrder, ErrProductDeleteIncompleteOrder) {
		t.Fatal("sentinels must differ")
	}
}

package model

import "testing"

func TestAllowedAdminTransition_shop(t *testing.T) {
	tests := []struct {
		from, to OrderStatus
		want     bool
	}{
		{OrderStatusPendingPayment, OrderStatusPaid, true},
		{OrderStatusPendingPayment, OrderStatusPacking, false},
		{OrderStatusPaid, OrderStatusPacking, true},
		{OrderStatusPaid, OrderStatusDelivered, true},
		{OrderStatusPacking, OrderStatusInTransit, true},
		{OrderStatusInTransit, OrderStatusDelivered, true},
		{OrderStatusDelivered, OrderStatusPacking, false},
		{orderStatusLegacyProcessing, OrderStatusInTransit, true},
		{OrderStatusAbandoned, OrderStatusPaid, false},
		{OrderStatusAbandoned, OrderStatusPacking, false},
		{OrderStatusCreated, OrderStatusPendingPayment, false},
		{OrderStatusCreated, OrderStatusRejected, false},
		{OrderStatusPendingPayment, OrderStatusRejected, false},
		{OrderStatusPaid, OrderStatusRejected, false},
	}

	for _, tc := range tests {
		got := AllowedAdminTransition(OrderTypeShop, tc.from, tc.to)
		if got != tc.want {
			t.Fatalf("shop %s -> %s: got %v want %v", tc.from, tc.to, got, tc.want)
		}
		// Empty order type must behave as shop.
		gotEmpty := AllowedAdminTransition("", tc.from, tc.to)
		if gotEmpty != tc.want {
			t.Fatalf("empty-type %s -> %s: got %v want %v", tc.from, tc.to, gotEmpty, tc.want)
		}
	}
}

func TestAllowedAdminTransition_custom(t *testing.T) {
	tests := []struct {
		from, to OrderStatus
		want     bool
	}{
		{OrderStatusCreated, OrderStatusPendingPayment, true},
		{OrderStatusCreated, OrderStatusRejected, true},
		{OrderStatusCreated, OrderStatusPaid, false},
		{OrderStatusRejected, OrderStatusPendingPayment, false},
		{OrderStatusRejected, OrderStatusPaid, false},
		{OrderStatusPendingPayment, OrderStatusPaid, true},
		{OrderStatusPendingPayment, OrderStatusRejected, false},
		{OrderStatusPaid, OrderStatusPacking, true},
		{OrderStatusPaid, OrderStatusDelivered, true},
		{OrderStatusPacking, OrderStatusInTransit, true},
		{OrderStatusInTransit, OrderStatusDelivered, true},
		{OrderStatusDelivered, OrderStatusPacking, false},
		{OrderStatusAbandoned, OrderStatusPaid, false},
	}

	for _, tc := range tests {
		got := AllowedAdminTransition(OrderTypeCustom, tc.from, tc.to)
		if got != tc.want {
			t.Fatalf("custom %s -> %s: got %v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestNormalizeOrderStatus_legacy(t *testing.T) {
	if OrderStatus("processing").Normalized() != OrderStatusPacking {
		t.Fatal("processing should normalize to packing")
	}
	if OrderStatus("shipped").Normalized() != OrderStatusInTransit {
		t.Fatal("shipped should normalize to in_transit")
	}
}

func TestNormalizeOrder_defaultsTypeToShop(t *testing.T) {
	o := &Order{Status: OrderStatusPaid}
	NormalizeOrder(o)
	if o.OrderType != OrderTypeShop {
		t.Fatalf("orderType = %q, want shop", o.OrderType)
	}
}

func TestNormalizeOrder_preservesCustom(t *testing.T) {
	o := &Order{OrderType: OrderTypeCustom, Status: OrderStatusCreated}
	NormalizeOrder(o)
	if o.OrderType != OrderTypeCustom {
		t.Fatalf("orderType = %q, want custom", o.OrderType)
	}
}

func TestOrderStatusAbandoned_validNotAdminSettable(t *testing.T) {
	if !OrderStatusAbandoned.Valid() {
		t.Fatal("abandoned should be a valid order status")
	}
	if OrderStatusAbandoned.AdminSettable() {
		t.Fatal("abandoned must not be admin-settable")
	}
	if _, err := ParseOrderStatus("abandoned"); err == nil {
		t.Fatal("ParseOrderStatus should reject abandoned as admin target")
	}
}

func TestOrderStatusCreatedAndRejected_validNotAdminSettable(t *testing.T) {
	for _, s := range []OrderStatus{OrderStatusCreated, OrderStatusRejected, OrderStatusPendingPayment} {
		if !s.Valid() {
			t.Fatalf("%s should be valid", s)
		}
		if s.AdminSettable() {
			t.Fatalf("%s must not be admin-settable via generic PATCH", s)
		}
		if _, err := ParseOrderStatus(string(s)); err == nil {
			t.Fatalf("ParseOrderStatus should reject %s as admin target", s)
		}
	}
}

func TestParseOrderListFilterStatus_allowsNewStatuses(t *testing.T) {
	for _, s := range []string{"abandoned", "pending_payment", "paid", "created", "rejected"} {
		got, err := ParseOrderListFilterStatus(s)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if string(got) != s {
			t.Fatalf("%q: got %q", s, got)
		}
	}
}

func TestAllowedAdminTransition_pendingToPaidStillAllowed(t *testing.T) {
	if !AllowedAdminTransition(OrderTypeShop, OrderStatusPendingPayment, OrderStatusPaid) {
		t.Fatal("admin should still mark shop pending_payment as paid")
	}
	if !AllowedAdminTransition(OrderTypeCustom, OrderStatusPendingPayment, OrderStatusPaid) {
		t.Fatal("admin should mark custom pending_payment as paid")
	}
}

func TestStatusChangeRequiresNote_rejectOnly(t *testing.T) {
	if !StatusChangeRequiresNote(OrderTypeCustom, OrderStatusCreated, OrderStatusRejected) {
		t.Fatal("custom reject requires note")
	}
	if StatusChangeRequiresNote(OrderTypeCustom, OrderStatusCreated, OrderStatusPendingPayment) {
		t.Fatal("custom accept should not require note")
	}
	if StatusChangeRequiresNote(OrderTypeShop, OrderStatusPendingPayment, OrderStatusPaid) {
		t.Fatal("shop paid should not require note")
	}
}

func TestValidateStatusChangeNote(t *testing.T) {
	err := ValidateStatusChangeNote(OrderTypeCustom, OrderStatusCreated, OrderStatusRejected, "  ")
	if err == nil {
		t.Fatal("expected error for blank reject reason")
	}
	if err := ValidateStatusChangeNote(OrderTypeCustom, OrderStatusCreated, OrderStatusRejected, "Out of capacity"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestParseOrderType(t *testing.T) {
	got, err := ParseOrderType("")
	if err != nil || got != OrderTypeShop {
		t.Fatalf("empty: got %q err %v", got, err)
	}
	got, err = ParseOrderType("custom")
	if err != nil || got != OrderTypeCustom {
		t.Fatalf("custom: got %q err %v", got, err)
	}
	if _, err := ParseOrderType("wholesale"); err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestMayAbandon(t *testing.T) {
	shopPending := &Order{OrderType: OrderTypeShop, Status: OrderStatusPendingPayment}
	if !shopPending.MayAbandon() {
		t.Fatal("shop pending_payment should allow abandon")
	}
	legacy := &Order{Status: OrderStatusPendingPayment}
	NormalizeOrder(legacy)
	if !legacy.MayAbandon() {
		t.Fatal("legacy shop-normalized pending should allow abandon")
	}
	customPending := &Order{OrderType: OrderTypeCustom, Status: OrderStatusPendingPayment}
	if customPending.MayAbandon() {
		t.Fatal("custom pending_payment must not allow abandon")
	}
	shopPaid := &Order{OrderType: OrderTypeShop, Status: OrderStatusPaid}
	if shopPaid.MayAbandon() {
		t.Fatal("shop paid must not allow abandon")
	}
}

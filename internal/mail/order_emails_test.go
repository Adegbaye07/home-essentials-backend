package mail_test

import (
	"strings"
	"testing"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
)

func TestOrderCustomerConfirmedEmail_greetsByName(t *testing.T) {
	order := &model.Order{
		TrackingNumber: "KAM-TEST",
		Customer:       model.CustomerInfo{Name: "Ada"},
	}
	_, html, plain, err := mail.OrderCustomerConfirmedEmail(order, "https://example.com/track")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hi Ada,") {
		t.Fatalf("html missing named greeting")
	}
	if !strings.Contains(plain, "Hi Ada,") {
		t.Fatalf("plain missing named greeting")
	}
}

func TestOrderStatusUpdateEmail_greetsByName(t *testing.T) {
	order := &model.Order{
		TrackingNumber: "KAM-TEST",
		Status:         model.OrderStatusPacking,
		Customer:       model.CustomerInfo{Name: "Ada"},
	}
	_, html, plain, err := mail.OrderStatusUpdateEmail(order, "", "https://example.com/track")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hi Ada,") {
		t.Fatalf("html missing named greeting")
	}
	if !strings.Contains(plain, "Hi Ada,") {
		t.Fatalf("plain missing named greeting")
	}
}

func TestOrderCustomerConfirmedEmail_emptyName(t *testing.T) {
	order := &model.Order{Customer: model.CustomerInfo{}}
	_, html, plain, err := mail.OrderCustomerConfirmedEmail(order, "https://example.com/track")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hi,") {
		t.Fatalf("html should fall back to Hi,")
	}
	if !strings.Contains(plain, "Hi,") {
		t.Fatalf("plain should fall back to Hi,")
	}
}

func TestOrderCustomerConfirmedEmail_totalsBreakdown(t *testing.T) {
	order := &model.Order{
		TrackingNumber:     "KAM-TEST",
		Customer:           model.CustomerInfo{Name: "Ada"},
		SubtotalAmountKobo: 500_000, // ₦5,000
		DeliveryFeeKobo:    200_000, // ₦2,000
		TotalAmountKobo:    700_000, // ₦7,000
		Items: []model.OrderItem{
			{ProductTitle: "Mat", LineTotalKobo: 500_000, Unit: model.OrderUnitPiece},
		},
	}
	_, html, plain, err := mail.OrderCustomerConfirmedEmail(order, "https://example.com/track")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Subtotal", "Delivery", "Total"} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(plain, "Subtotal:") || !strings.Contains(plain, "Delivery:") {
		t.Fatalf("plain missing amount breakdown: %s", plain)
	}
}

func TestOrderAdminPaidEmail_totalsBreakdownFreeDelivery(t *testing.T) {
	order := &model.Order{
		TrackingNumber:     "KAM-TEST",
		Customer:           model.CustomerInfo{Name: "Ada", Email: "a@b.com", Phone: "1", DeliveryAddress: "Addr"},
		SubtotalAmountKobo: 1_000_000, // ₦10,000
		DeliveryFeeKobo:    0,
		TotalAmountKobo:    1_000_000,
		Items: []model.OrderItem{
			{ProductTitle: "Mat", LineTotalKobo: 1_000_000, Unit: model.OrderUnitBundle},
		},
	}
	_, html, plain, err := mail.OrderAdminPaidEmail(order)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Free") {
		t.Fatalf("html should show Free delivery")
	}
	if !strings.Contains(plain, "Delivery: Free") {
		t.Fatalf("plain should show Free delivery: %s", plain)
	}
}

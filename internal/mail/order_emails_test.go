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

func TestOrderCustomCreatedEmail(t *testing.T) {
	order := &model.Order{
		TrackingNumber: "KAM-CUSTOM",
		Customer:       model.CustomerInfo{Name: "Ada"},
		Custom: &model.CustomRequest{
			Title:            "Tote",
			Description:      "Canvas tote with lining",
			Sizes:            []string{"2 x 5 ft", "3 x 5 ft"},
			Colors:           []string{"olive", "cream"},
			Quantity:         3,
			OfferedTotalKobo: 4500000,
		},
	}
	_, html, plain, err := mail.OrderCustomCreatedEmail(order, "https://example.com/track")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hi Ada,") {
		t.Fatalf("html missing greeting")
	}
	if !strings.Contains(html, "KAM-CUSTOM") || !strings.Contains(plain, "KAM-CUSTOM") {
		t.Fatalf("missing tracking id")
	}
	if !strings.Contains(html, "Tote") || !strings.Contains(plain, "olive") {
		t.Fatalf("missing custom details")
	}
}

func TestOrderCustomCreatedEmail_requiresCustom(t *testing.T) {
	_, _, _, err := mail.OrderCustomCreatedEmail(&model.Order{Customer: model.CustomerInfo{}}, "")
	if err == nil {
		t.Fatal("expected error when custom is nil")
	}
}

func TestOrderCustomPaymentLinkEmail_acceptAndResend(t *testing.T) {
	order := &model.Order{
		TrackingNumber:  "KAM-X",
		TotalAmountKobo: 2500000,
		Customer:        model.CustomerInfo{Name: "Ada"},
		Custom:          &model.CustomRequest{Title: "Tote"},
	}
	subject, html, plain, err := mail.OrderCustomPaymentLinkEmail(order, "https://pay.example/link", "https://shop/track", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(subject, "accepted") {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(html, "Pay now") || !strings.Contains(plain, "https://pay.example/link") {
		t.Fatalf("missing payment link content")
	}
	subject2, html2, _, err := mail.OrderCustomPaymentLinkEmail(order, "https://pay.example/link2", "https://shop/track", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(subject2, "new payment link") {
		t.Fatalf("resent subject = %q", subject2)
	}
	if !strings.Contains(html2, "fresh payment link") {
		t.Fatalf("resent copy missing")
	}
}

func TestOrderCustomPaymentLinkEmail_requiresURL(t *testing.T) {
	order := &model.Order{
		TrackingNumber:  "KAM-X",
		TotalAmountKobo: 100,
		Customer:        model.CustomerInfo{Name: "Ada"},
	}
	_, _, _, err := mail.OrderCustomPaymentLinkEmail(order, "", "https://track", false)
	if err == nil {
		t.Fatal("expected error for empty payment url")
	}
}

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

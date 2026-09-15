package controller

import (
	"strings"
	"testing"
)

func TestValidateCustomer_requiresName(t *testing.T) {
	err := validateCustomer(CustomerInput{
		Email:           "a@b.com",
		Phone:           "0800",
		DeliveryAddress: "Addr",
	})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name required, got %v", err)
	}
}

func TestValidateCustomer_ok(t *testing.T) {
	err := validateCustomer(CustomerInput{
		Name:            " Ada ",
		Email:           "a@b.com",
		Phone:           "0800",
		DeliveryAddress: "Addr",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeCustomer(CustomerInput{
		Name:            " Ada ",
		Email:           " a@b.com ",
		Phone:           " 0800 ",
		DeliveryAddress: " Addr ",
	})
	if got.Name != "Ada" {
		t.Fatalf("name not trimmed: %q", got.Name)
	}
}

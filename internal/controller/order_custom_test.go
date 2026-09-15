package controller

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestValidateCustomRequest_ok(t *testing.T) {
	got, err := validateCustomRequest(CustomRequestInput{
		Title:            " Custom rug ",
		Description:      " Soft pile ",
		Sizes:            []string{"2 x 5 ft", "3 x 5 ft", "2 x 5 ft"},
		Colors:           []string{" Black ", "tan", "BLACK"},
		Quantity:         2,
		OfferedTotalKobo: 15000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Custom rug" {
		t.Fatalf("title = %q", got.Title)
	}
	if len(got.Sizes) != 2 || got.Sizes[0] != "2 x 5 ft" || got.Sizes[1] != "3 x 5 ft" {
		t.Fatalf("sizes = %#v", got.Sizes)
	}
	if len(got.Colors) != 2 || got.Colors[0] != "Black" || got.Colors[1] != "tan" {
		t.Fatalf("colors = %#v", got.Colors)
	}
}

func TestValidateCustomRequest_rejectsBadInput(t *testing.T) {
	base := CustomRequestInput{
		Title:            "Item",
		Description:      "Desc",
		Sizes:            []string{"2ft"},
		Colors:           []string{"black"},
		Quantity:         1,
		OfferedTotalKobo: 100,
	}

	cases := []struct {
		name string
		mut  func(*CustomRequestInput)
		want string
	}{
		{"empty title", func(in *CustomRequestInput) { in.Title = "  " }, "title"},
		{"empty description", func(in *CustomRequestInput) { in.Description = "" }, "description"},
		{"qty zero", func(in *CustomRequestInput) { in.Quantity = 0 }, "quantity"},
		{"offered zero", func(in *CustomRequestInput) { in.OfferedTotalKobo = 0 }, "offeredTotalKobo"},
		{"no sizes", func(in *CustomRequestInput) { in.Sizes = nil }, "size"},
		{"blank sizes", func(in *CustomRequestInput) { in.Sizes = []string{" ", ""} }, "size"},
		{"no colors", func(in *CustomRequestInput) { in.Colors = []string{" ", ""} }, "color"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mut(&in)
			_, err := validateCustomRequest(in)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCreateCustom_requiresUploader(t *testing.T) {
	ctrl := NewOrderController(nil, nil, nil, nil, nil, slog.Default(), "http://localhost:3000")
	_, err := ctrl.CreateCustom(context.Background(), CreateCustomOrderInput{
		Customer: CustomerInput{
			Name: "Ada", Email: "a@b.com", Phone: "0800", DeliveryAddress: "Lagos",
		},
		Custom: CustomRequestInput{
			Title: "Item", Description: "Desc", Sizes: []string{"2ft"},
			Colors: []string{"black"}, Quantity: 1, OfferedTotalKobo: 1000,
		},
	}, CustomSampleImage{
		Filename: "x.jpg", ContentType: "image/jpeg", Reader: strings.NewReader("fake"),
	})
	if err != ErrUploadNotConfigured {
		t.Fatalf("got %v, want ErrUploadNotConfigured", err)
	}
}

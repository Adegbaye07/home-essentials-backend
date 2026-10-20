package controller

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestValidateCustomRequest_ok(t *testing.T) {
	got, err := validateCustomRequest(CustomRequestInput{
		Title:    " Multipurpose hallway mat ",
		Quantity: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Multipurpose hallway mat" {
		t.Fatalf("title = %q", got.Title)
	}
	if got.Quantity != 3 {
		t.Fatalf("quantity = %d", got.Quantity)
	}
}

func TestValidateCustomRequest_rejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		in   CustomRequestInput
		want string
	}{
		{"empty title", CustomRequestInput{Title: "  ", Quantity: 1}, "title"},
		{"qty zero", CustomRequestInput{Title: "Mat", Quantity: 0}, "quantity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateCustomRequest(tc.in)
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
	ctrl := NewOrderController(nil, nil, nil, nil, nil, "", slog.Default(), "http://localhost:3000")
	_, err := ctrl.CreateCustom(context.Background(), CreateCustomOrderInput{
		Customer: CustomerInput{
			Name: "Ada", Email: "a@b.com", Phone: "0800", DeliveryAddress: "Lagos",
		},
		Custom: CustomRequestInput{Title: "Mat", Quantity: 1},
	}, CustomSampleImage{
		Filename: "x.jpg", ContentType: "image/jpeg", Reader: strings.NewReader("fake"),
	})
	if err != ErrUploadNotConfigured {
		t.Fatalf("got %v, want ErrUploadNotConfigured", err)
	}
}

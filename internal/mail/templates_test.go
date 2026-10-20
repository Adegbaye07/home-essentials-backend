package mail_test

import (
	"strings"
	"testing"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
)

func TestRenderDemoIncludesBrandAndFooter(t *testing.T) {
	data := mail.DemoEmailData{
		ShellData: mail.NewShellData("Test", "Hello"),
		Intro:     "Preview line.",
		ItemsTableData: mail.ItemsTableData{
			Items: []mail.LineItemRow{
				{
					ProductTitle: "Door mat",
					Size:         "2 x 3 ft",
					Variant:      "brown",
					Unit:         "piece",
					Quantity:     2,
					UnitPrice:    mail.FormatNGN(1800000),
					LineTotal:    mail.FormatNGN(3600000),
				},
			},
		},
	}
	html, err := mail.Render("demo", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Home Essentials by Kamgol",
		"Door mat",
		"brown",
		"do not reply",
		"Preview line.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
}

func TestOrderCustomEmailsRender(t *testing.T) {
	order := &model.Order{
		TrackingNumber:  "KAM-20260101-ABC123",
		TotalAmountKobo: 2500000,
		Customer: model.CustomerInfo{
			Name: "Ada", Email: "a@b.com", Phone: "0800", DeliveryAddress: "Lagos",
		},
		Custom: &model.CustomRequest{
			Title: "Hallway mat", Quantity: 2, SampleImageURL: "https://example.com/s.jpg",
		},
	}

	_, html, _, err := mail.OrderCustomCreatedEmail(order, "https://shop.example/track")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hallway mat") ||
		!strings.Contains(html, "KAM-20260101-ABC123") ||
		!strings.Contains(html, "View sample image") ||
		!strings.Contains(html, "https://example.com/s.jpg") {
		t.Fatalf("created email missing fields: %s", html)
	}

	_, adminHTML, _, err := mail.OrderCustomAdminNotifyEmail(order)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(adminHTML, "a@b.com") || !strings.Contains(adminHTML, "View sample") {
		t.Fatalf("admin notify missing fields: %s", adminHTML)
	}

	_, payHTML, _, err := mail.OrderCustomPaymentLinkEmail(order, "https://checkout.paystack.com/x", "https://shop.example/track", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payHTML, "Pay now") || !strings.Contains(payHTML, "includes delivery") {
		t.Fatalf("payment link email missing fields: %s", payHTML)
	}
}

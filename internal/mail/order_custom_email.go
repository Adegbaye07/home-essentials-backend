package mail

import (
	"fmt"
	"strings"

	"homeessentials/backend/internal/model"
)

type OrderCustomCreatedData struct {
	ShellData
	CustomerName   string
	TrackingNumber string
	TrackPageURL   string
	Title          string
	Description    string
	Sizes          string
	Colors         string
	Quantity       int
	OfferedTotal   string
}

// OrderCustomCreatedEmail acknowledges a bespoke recreate-order submission.
func OrderCustomCreatedEmail(order *model.Order, trackPageURL string) (subject, html, plain string, err error) {
	if order == nil || order.Custom == nil {
		return "", "", "", fmt.Errorf("custom order details are required")
	}

	subject = "We received your Home Essentials by Kamgol custom request"
	shell := NewShellData(subject, "Let's make a bag")

	sizes := make([]string, 0, len(order.Custom.Sizes))
	for _, s := range order.Custom.Sizes {
		sizes = append(sizes, string(s))
	}

	data := OrderCustomCreatedData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(order.Customer.Name),
		TrackingNumber: order.TrackingNumber,
		TrackPageURL:   trackPageURL,
		Title:          order.Custom.Title,
		Description:    order.Custom.Description,
		Sizes:          strings.Join(sizes, ", "),
		Colors:         strings.Join(order.Custom.Colors, ", "),
		Quantity:       order.Custom.Quantity,
		OfferedTotal:   FormatNGN(order.Custom.OfferedTotalKobo),
	}
	html, err = Render("order-custom-created", data)
	if err != nil {
		return "", "", "", err
	}

	plain = fmt.Sprintf(
		"%s\n\nWe received your request.\n\nTracking: %s\nTitle: %s\nSizes: %s\nColors: %s\nQuantity: %d\nOffered total: %s\n\nTrack: %s\n",
		greetingLine(order.Customer.Name),
		order.TrackingNumber,
		order.Custom.Title,
		data.Sizes,
		data.Colors,
		order.Custom.Quantity,
		data.OfferedTotal,
		trackPageURL,
	)
	return subject, html, plain, nil
}

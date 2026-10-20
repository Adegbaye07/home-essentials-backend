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
	Quantity       int
	SampleImageURL string
}

// OrderCustomCreatedEmail acknowledges a custom mat-order submission.
func OrderCustomCreatedEmail(order *model.Order, trackPageURL string) (subject, html, plain string, err error) {
	if order == nil || order.Custom == nil {
		return "", "", "", fmt.Errorf("custom order details are required")
	}

	subject = "We received your Home Essentials by Kamgol custom request"
	shell := NewShellData(subject, "Request received")

	data := OrderCustomCreatedData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(order.Customer.Name),
		TrackingNumber: order.TrackingNumber,
		TrackPageURL:   trackPageURL,
		Title:          order.Custom.Title,
		Quantity:       order.Custom.Quantity,
		SampleImageURL: order.Custom.SampleImageURL,
	}
	html, err = Render("order-custom-created", data)
	if err != nil {
		return "", "", "", err
	}

	plain = fmt.Sprintf(
		"%s\n\nWe received your custom mat request.\n\nTracking: %s\nTitle: %s\nQuantity: %d\n",
		greetingLine(order.Customer.Name),
		order.TrackingNumber,
		order.Custom.Title,
		order.Custom.Quantity,
	)
	if order.Custom.SampleImageURL != "" {
		plain += fmt.Sprintf("Sample: %s\n", order.Custom.SampleImageURL)
	}
	plain += fmt.Sprintf("\nTrack: %s\n", trackPageURL)
	return subject, html, plain, nil
}

type OrderCustomAdminNotifyData struct {
	ShellData
	TrackingNumber  string
	Title           string
	Quantity        int
	CustomerName    string
	CustomerEmail   string
	CustomerPhone   string
	DeliveryAddress string
	SampleImageURL  string
}	

// OrderCustomAdminNotifyEmail alerts the shop of a new custom mat request.
func OrderCustomAdminNotifyEmail(order *model.Order) (subject, html, plain string, err error) {
	if order == nil || order.Custom == nil {
		return "", "", "", fmt.Errorf("custom order details are required")
	}

	subject = "Home Essentials by Kamgol: new custom mat request"
	shell := NewShellData(subject, "New custom mat request")

	data := OrderCustomAdminNotifyData{
		ShellData:       shell,
		TrackingNumber:  order.TrackingNumber,
		Title:           order.Custom.Title,
		Quantity:        order.Custom.Quantity,
		CustomerName:    strings.TrimSpace(order.Customer.Name),
		CustomerEmail:   order.Customer.Email,
		CustomerPhone:   order.Customer.Phone,
		DeliveryAddress: order.Customer.DeliveryAddress,
		SampleImageURL:  order.Custom.SampleImageURL,
	}
	html, err = Render("order-custom-admin-notify", data)
	if err != nil {
		return "", "", "", err
	}

	plain = fmt.Sprintf(
		"New custom mat request\n\nTracking: %s\nTitle: %s\nQuantity: %d\nCustomer: %s <%s>\nPhone: %s\nAddress: %s\nSample: %s\n",
		order.TrackingNumber,
		order.Custom.Title,
		order.Custom.Quantity,
		order.Customer.Name,
		order.Customer.Email,
		order.Customer.Phone,
		order.Customer.DeliveryAddress,
		order.Custom.SampleImageURL,
	)
	return subject, html, plain, nil
}

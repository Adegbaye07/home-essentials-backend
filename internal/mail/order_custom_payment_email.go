package mail

import (
	"fmt"
	"strings"

	"homeessentials/backend/internal/model"
)

type OrderCustomPaymentLinkData struct {
	ShellData
	CustomerName   string
	TrackingNumber string
	TrackPageURL   string
	Title          string
	AmountDue      string
	PaymentURL     string
	Resent         bool
}

// OrderCustomPaymentLinkEmail emails the Paystack hosted checkout link after accept or resend.
func OrderCustomPaymentLinkEmail(order *model.Order, paymentURL, trackPageURL string, resent bool) (subject, html, plain string, err error) {
	if order == nil {
		return "", "", "", fmt.Errorf("order is required")
	}
	paymentURL = strings.TrimSpace(paymentURL)
	if paymentURL == "" {
		return "", "", "", fmt.Errorf("payment url is required")
	}

	if resent {
		subject = "Home Essentials by Kamgol: new payment link for your custom request"
	} else {
		subject = "Home Essentials by Kamgol: your custom request was accepted — pay now"
	}
	heading := "Payment link"
	if !resent {
		heading = "Request accepted"
	}
	shell := NewShellData(subject, heading)

	title := ""
	if order.Custom != nil {
		title = order.Custom.Title
	}

	data := OrderCustomPaymentLinkData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(order.Customer.Name),
		TrackingNumber: order.TrackingNumber,
		TrackPageURL:   trackPageURL,
		Title:          title,
		AmountDue:      FormatNGN(order.TotalAmountKobo),
		PaymentURL:     paymentURL,
		Resent:         resent,
	}
	html, err = Render("order-custom-payment-link", data)
	if err != nil {
		return "", "", "", err
	}

	intro := "We can fulfill your custom bag request. Please complete payment:"
	if resent {
		intro = "Here is a fresh payment link for your custom bag request:"
	}
	plain = fmt.Sprintf(
		"%s\n\n%s\n\nTracking: %s\nAmount due: %s\nPay: %s\nTrack: %s\n",
		greetingLine(order.Customer.Name),
		intro,
		order.TrackingNumber,
		data.AmountDue,
		paymentURL,
		trackPageURL,
	)
	return subject, html, plain, nil
}

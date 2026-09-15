package mail

import (
	"fmt"
	"strings"

	"homeessentials/backend/internal/model"
)

func statusLabel(s model.OrderStatus) string {
	switch s.Normalized() {
	case model.OrderStatusCreated:
		return "Created"
	case model.OrderStatusPendingPayment:
		return "Pending payment"
	case model.OrderStatusAbandoned:
		return "Abandoned"
	case model.OrderStatusRejected:
		return "Rejected"
	case model.OrderStatusPaid:
		return "Paid"
	case model.OrderStatusPacking:
		return "Packing"
	case model.OrderStatusInTransit:
		return "In transit"
	case model.OrderStatusDelivered:
		return "Delivered"
	default:
		return string(s)
	}
}

type OrderStatusUpdateData struct {
	ShellData
	CustomerName   string
	TrackingNumber string
	StatusLabel    string
	Note           string
	TrackPageURL   string
}

// OrderStatusUpdateEmail builds HTML (and plain) for an admin status change.
func OrderStatusUpdateEmail(order *model.Order, note, trackPageURL string) (subject, html, plain string, err error) {
	label := statusLabel(order.Status)
	subject = fmt.Sprintf("Home Essentials by Kamgol order update: %s", label)
	shell := NewShellData(subject, "Order update")

	data := OrderStatusUpdateData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(order.Customer.Name),
		TrackingNumber: order.TrackingNumber,
		StatusLabel:    label,
		Note:           note,
		TrackPageURL:   trackPageURL,
	}
	html, err = Render("order-status-update", data)
	if err != nil {
		return "", "", "", err
	}

	plain = fmt.Sprintf(
		"%s\n\nYour Home Essentials by Kamgol order status is now: %s.\n\n",
		greetingLine(order.Customer.Name),
		label,
	)
	if order.TrackingNumber != "" {
		plain += fmt.Sprintf("Tracking ID: %s\n", order.TrackingNumber)
	}
	if note != "" {
		plain += fmt.Sprintf("\nNote from our team: %s\n", note)
	}
	if trackPageURL != "" && order.TrackingNumber != "" {
		plain += fmt.Sprintf("\nTrack your order: %s\n", trackPageURL)
	}
	plain += "\nThank you for shopping with Home Essentials by Kamgol.\n"
	return subject, html, plain, nil
}

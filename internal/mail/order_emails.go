package mail

import (
	"fmt"
	"strings"

	"homeessentials/backend/internal/model"
)

// TotalsBlockData feeds the totals-block partial.
type TotalsBlockData struct {
	TotalPaid string
}

type OrderCustomerConfirmedData struct {
	ShellData
	ItemsTableData
	TotalsBlockData
	CustomerName   string
	TrackingNumber string
	TrackPageURL   string
}

type OrderAdminPaidData struct {
	ShellData
	ItemsTableData
	TotalsBlockData
	OrderID         string
	TrackingNumber  string
	CustomerName    string
	CustomerEmail   string
	CustomerPhone   string
	DeliveryAddress string
}

func lineItemsFromOrder(order *model.Order) []LineItemRow {
	rows := make([]LineItemRow, 0, len(order.Items))
	for _, it := range order.Items {
		rows = append(rows, LineItemRow{
			ProductTitle: it.ProductTitle,
			Size:         string(it.Size),
			Color:        it.Color,
			Quantity:     it.Quantity,
			UnitPrice:    FormatNGN(it.UnitPriceKobo),
			LineTotal:    FormatNGN(it.LineTotalKobo),
		})
	}
	return rows
}

func totalsFromOrder(order *model.Order) TotalsBlockData {
	return TotalsBlockData{
		TotalPaid: FormatNGN(order.TotalAmountKobo),
	}
}

// OrderCustomerConfirmedEmail renders the post-payment customer email.
func OrderCustomerConfirmedEmail(order *model.Order, trackPageURL string) (subject, html, plain string, err error) {
	subject = "Your Home Essentials by Kamgol order is confirmed"
	shell := NewShellData(subject, "Order confirmed")
	data := OrderCustomerConfirmedData{
		ShellData:       shell,
		ItemsTableData:  ItemsTableData{Items: lineItemsFromOrder(order)},
		TotalsBlockData: totalsFromOrder(order),
		CustomerName:    strings.TrimSpace(order.Customer.Name),
		TrackingNumber:  order.TrackingNumber,
		TrackPageURL:    trackPageURL,
	}
	html, err = Render("order-customer-confirmed", data)
	if err != nil {
		return "", "", "", err
	}
	t := totalsFromOrder(order)
	plain = fmt.Sprintf(
		"%s\n\nThank you for your Home Essentials by Kamgol order.\n\nTracking: %s\nTotal: %s\n\nTrack: %s\n",
		greetingLine(order.Customer.Name),
		order.TrackingNumber,
		t.TotalPaid,
		trackPageURL,
	)
	return subject, html, plain, nil
}

// OrderAdminPaidEmail renders the admin notification for a paid order.
func OrderAdminPaidEmail(order *model.Order) (subject, html, plain string, err error) {
	subject = "Home Essentials by Kamgol: order paid"
	shell := NewShellData(subject, "New paid order")
	data := OrderAdminPaidData{
		ShellData:       shell,
		ItemsTableData:  ItemsTableData{Items: lineItemsFromOrder(order)},
		TotalsBlockData: totalsFromOrder(order),
		OrderID:         order.ID.Hex(),
		TrackingNumber:  order.TrackingNumber,
		CustomerName:    strings.TrimSpace(order.Customer.Name),
		CustomerEmail:   order.Customer.Email,
		CustomerPhone:   order.Customer.Phone,
		DeliveryAddress: order.Customer.DeliveryAddress,
	}
	html, err = Render("order-admin-paid", data)
	if err != nil {
		return "", "", "", err
	}
	t := totalsFromOrder(order)
	who := strings.TrimSpace(order.Customer.Name)
	if who == "" {
		who = order.Customer.Email
	} else {
		who = who + " (" + order.Customer.Email + ")"
	}
	plain = fmt.Sprintf(
		"New paid order %s\nCustomer: %s (%s)\nAddress: %s\nTotal: %s\nTracking: %s\n",
		order.ID.Hex(),
		who,
		order.Customer.Phone,
		order.Customer.DeliveryAddress,
		t.TotalPaid,
		order.TrackingNumber,
	)
	return subject, html, plain, nil
}

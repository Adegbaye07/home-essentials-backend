package model

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderType string

const (
	OrderTypeShop   OrderType = "shop"
	OrderTypeCustom OrderType = "custom"
)

func (t OrderType) Valid() bool {
	switch t {
	case OrderTypeShop, OrderTypeCustom:
		return true
	default:
		return false
	}
}

// Normalized maps empty/legacy missing types to shop (existing Mongo documents).
func (t OrderType) Normalized() OrderType {
	if t == "" {
		return OrderTypeShop
	}
	return t
}

func ParseOrderType(s string) (OrderType, error) {
	raw := OrderType(strings.TrimSpace(s))
	if raw == "" {
		return OrderTypeShop, nil
	}
	if !raw.Valid() {
		return "", fmt.Errorf("invalid order type %q", s)
	}
	return raw, nil
}

type OrderStatus string

const (
	OrderStatusCreated        OrderStatus = "created"
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusAbandoned      OrderStatus = "abandoned"
	OrderStatusRejected       OrderStatus = "rejected"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusPacking        OrderStatus = "packing"
	OrderStatusInTransit      OrderStatus = "in_transit"
	OrderStatusDelivered      OrderStatus = "delivered"
)

// Legacy values kept for normalizing old Mongo documents.
const (
	orderStatusLegacyProcessing OrderStatus = "processing"
	orderStatusLegacyShipped    OrderStatus = "shipped"
	orderStatusLegacyCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) Valid() bool {
	switch s {
	case OrderStatusCreated, OrderStatusPendingPayment, OrderStatusAbandoned, OrderStatusRejected,
		OrderStatusPaid, OrderStatusPacking, OrderStatusInTransit, OrderStatusDelivered:
		return true
	default:
		return false
	}
}

// Normalized maps legacy status strings to the current enum (in-place on orders read from DB).
func (s OrderStatus) Normalized() OrderStatus {
	switch s {
	case orderStatusLegacyProcessing:
		return OrderStatusPacking
	case orderStatusLegacyShipped:
		return OrderStatusInTransit
	default:
		return s
	}
}

func ParseOrderStatus(s string) (OrderStatus, error) {
	status := OrderStatus(s).Normalized()
	if !status.Valid() {
		return "", fmt.Errorf("invalid order status %q", s)
	}
	if !status.AdminSettable() {
		return "", fmt.Errorf("invalid order status %q", s)
	}
	return status, nil
}

// ParseOrderListFilterStatus parses a status query for admin order list filters.
func ParseOrderListFilterStatus(s string) (OrderStatus, error) {
	status := OrderStatus(s).Normalized()
	if !status.Valid() {
		return "", fmt.Errorf("invalid order status %q", s)
	}
	return status, nil
}

// AdminSettable reports whether an admin generic PATCH may set an order to this status.
// created / rejected / pending_payment / abandoned use dedicated flows (accept, reject, payment).
func (s OrderStatus) AdminSettable() bool {
	switch s.Normalized() {
	case OrderStatusPaid, OrderStatusPacking, OrderStatusInTransit, OrderStatusDelivered:
		return true
	default:
		return false
	}
}

// AllowedAdminTransition reports whether an admin may move an order of the given type from current to next.
// Accept (created → pending_payment) and reject (created → rejected) for custom orders are included
// here so dedicated endpoints can share the same rules; generic PATCH still blocks non-AdminSettable targets.
func AllowedAdminTransition(orderType OrderType, current, next OrderStatus) bool {
	orderType = orderType.Normalized()
	current = current.Normalized()
	next = next.Normalized()
	if !next.Valid() {
		return false
	}
	if current == next {
		return false
	}
	if current == OrderStatusDelivered || current == OrderStatusRejected {
		return false
	}
	if next == OrderStatusAbandoned {
		return false
	}
	// Shop orders never use custom-only statuses.
	if orderType != OrderTypeCustom {
		if current == OrderStatusCreated || current == OrderStatusRejected {
			return false
		}
		if next == OrderStatusCreated || next == OrderStatusRejected {
			return false
		}
	}
	return slices.Contains(adminNextStatuses(orderType, current), next)
}

func adminNextStatuses(orderType OrderType, current OrderStatus) []OrderStatus {
	switch current {
	case OrderStatusCreated:
		if orderType == OrderTypeCustom {
			return []OrderStatus{OrderStatusPendingPayment, OrderStatusRejected}
		}
		return nil
	case OrderStatusPendingPayment:
		return []OrderStatus{OrderStatusPaid}
	case OrderStatusPaid:
		return []OrderStatus{OrderStatusPacking, OrderStatusInTransit, OrderStatusDelivered}
	case OrderStatusPacking:
		return []OrderStatus{OrderStatusInTransit, OrderStatusDelivered}
	case OrderStatusInTransit:
		return []OrderStatus{OrderStatusDelivered}
	default:
		return nil
	}
}

// StatusChangeRequiresNote reports whether a transition must include a non-empty admin note.
func StatusChangeRequiresNote(orderType OrderType, current, next OrderStatus) bool {
	_ = current
	return orderType.Normalized() == OrderTypeCustom && next.Normalized() == OrderStatusRejected
}

// ValidateStatusChangeNote returns an error when a required note is missing.
func ValidateStatusChangeNote(orderType OrderType, current, next OrderStatus, note string) error {
	if !StatusChangeRequiresNote(orderType, current, next) {
		return nil
	}
	if strings.TrimSpace(note) == "" {
		return fmt.Errorf("a reason is required when rejecting a custom order")
	}
	return nil
}

func NormalizeOrder(o *Order) {
	if o == nil {
		return
	}
	o.OrderType = o.OrderType.Normalized()
	o.Status = o.Status.Normalized()
	for i := range o.StatusHistory {
		o.StatusHistory[i].Status = o.StatusHistory[i].Status.Normalized()
	}
}

// MayAbandon reports whether payment abandon is allowed (shop checkout cancel only).
func (o *Order) MayAbandon() bool {
	if o == nil {
		return false
	}
	return o.OrderType.Normalized() == OrderTypeShop && o.Status.Normalized() == OrderStatusPendingPayment
}

// CustomRequest holds bespoke recreate-order details (orderType=custom).
type CustomRequest struct {
	Title            string     `bson:"title" json:"title"`
	Description      string     `bson:"description" json:"description"`
	Sizes            []SizeCode `bson:"sizes" json:"sizes"`
	Colors           []string   `bson:"colors" json:"colors"`
	Quantity         int        `bson:"quantity" json:"quantity"`
	OfferedTotalKobo int64      `bson:"offeredTotalKobo" json:"offeredTotalKobo"`
	SampleImageURL   string     `bson:"sampleImageUrl,omitempty" json:"sampleImageUrl,omitempty"`
}

type OrderItem struct {
	ProductID     primitive.ObjectID `bson:"productId" json:"productId"`
	ProductTitle  string             `bson:"productTitle" json:"productTitle"`
	Size          SizeCode           `bson:"size" json:"size"`
	Color         string             `bson:"color" json:"color"`
	ImageURL      string             `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	Quantity      int                `bson:"quantity" json:"quantity"`
	UnitPriceKobo int64              `bson:"unitPriceKobo" json:"unitPriceKobo"`
	LineTotalKobo int64              `bson:"lineTotalKobo" json:"lineTotalKobo"`
}

type CustomerInfo struct {
	Name            string `bson:"name,omitempty" json:"name"`
	Email           string `bson:"email" json:"email"`
	Phone           string `bson:"phone" json:"phone"`
	DeliveryAddress string `bson:"deliveryAddress" json:"deliveryAddress"`
}

type StatusHistoryEntry struct {
	Status OrderStatus `bson:"status" json:"status"`
	At     time.Time   `bson:"at" json:"at"`
	Note   string      `bson:"note,omitempty" json:"note,omitempty"`
}

type OrderNotifications struct {
	EmailedAt *time.Time `bson:"emailedAt,omitempty" json:"emailedAt,omitempty"`
}

type Order struct {
	ID                primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	OrderType         OrderType            `bson:"orderType,omitempty" json:"orderType"`
	Items             []OrderItem          `bson:"items" json:"items"`
	Custom            *CustomRequest       `bson:"custom,omitempty" json:"custom,omitempty"`
	Customer          CustomerInfo         `bson:"customer" json:"customer"`
	Status            OrderStatus          `bson:"status" json:"status"`
	StatusHistory     []StatusHistoryEntry `bson:"statusHistory" json:"statusHistory"`
	TotalAmountKobo   int64                `bson:"totalAmountKobo" json:"totalAmountKobo"`
	PaystackReference string               `bson:"paystackReference" json:"paystackReference"`
	TrackingNumber    string               `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	PaidAt            *time.Time           `bson:"paidAt,omitempty" json:"paidAt,omitempty"`
	AbandonedAt       *time.Time           `bson:"abandonedAt,omitempty" json:"abandonedAt,omitempty"`
	Notifications     OrderNotifications   `bson:"notifications" json:"notifications"`
	CreatedAt         time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time            `bson:"updatedAt" json:"updatedAt"`
}

package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/paystack"
	"homeessentials/backend/internal/repository"
)

var (
	ErrAbandonEmailMismatch = errors.New("email does not match order")
	ErrAbandonConflict      = errors.New("order cannot be abandoned")
)

func IsAbandonEmailMismatch(err error) bool {
	return errors.Is(err, ErrAbandonEmailMismatch)
}

func IsAbandonConflict(err error) bool {
	return errors.Is(err, ErrAbandonConflict)
}

type PaymentController struct {
	orders          *repository.OrderRepository
	paystack        *paystack.Client
	mail            *mail.Sender
	adminEmail      string
	clientPublicURL string
	log             *slog.Logger
}

func NewPaymentController(
	orders *repository.OrderRepository,
	ps *paystack.Client,
	mailer *mail.Sender,
	adminEmail string,
	clientPublicURL string,
	log *slog.Logger,
) *PaymentController {
	return &PaymentController{
		orders:          orders,
		paystack:        ps,
		mail:            mailer,
		adminEmail:      adminEmail,
		clientPublicURL: strings.TrimSuffix(clientPublicURL, "/"),
		log:             log,
	}
}

type InitializePaymentResult struct {
	AccessCode       string `json:"accessCode"`
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	Reference        string `json:"reference"`
}

func (c *PaymentController) Initialize(ctx context.Context, orderID primitive.ObjectID) (InitializePaymentResult, error) {
	if c.paystack == nil {
		return InitializePaymentResult{}, fmt.Errorf("paystack is not configured")
	}

	order, err := c.orders.FindByID(ctx, orderID)
	if err != nil {
		return InitializePaymentResult{}, err
	}
	model.NormalizeOrder(order)

	switch order.Status {
	case model.OrderStatusPendingPayment:
		// ok — shop checkout
	case model.OrderStatusAbandoned:
		now := time.Now().UTC()
		order.Status = model.OrderStatusPendingPayment
		order.AbandonedAt = nil
		order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
			Status: model.OrderStatusPendingPayment,
			At:     now,
			Note:   "Payment resumed",
		})
		if err := c.orders.Update(ctx, order); err != nil {
			return InitializePaymentResult{}, err
		}
	default:
		return InitializePaymentResult{}, fmt.Errorf("order is not awaiting payment")
	}

	init, err := c.paystack.InitializeTransaction(ctx, order.Customer.Email, order.TotalAmountKobo, order.PaystackReference)
	if err != nil {
		return InitializePaymentResult{}, fmt.Errorf("initialize paystack: %w", err)
	}

	if init.Reference != "" && init.Reference != order.PaystackReference {
		order.PaystackReference = init.Reference
		if err := c.orders.Update(ctx, order); err != nil {
			return InitializePaymentResult{}, err
		}
	}

	return InitializePaymentResult{
		AccessCode:       init.AccessCode,
		AuthorizationURL: init.AuthorizationURL,
		Reference:        order.PaystackReference,
	}, nil
}

type VerifyPaymentResult struct {
	OrderID        string            `json:"orderId"`
	Status         model.OrderStatus `json:"status"`
	TrackingNumber string            `json:"trackingNumber,omitempty"`
	PaystackStatus string            `json:"paystackStatus,omitempty"`
	CustomerEmail  string            `json:"customerEmail"`
}

func (c *PaymentController) Verify(ctx context.Context, reference string) (VerifyPaymentResult, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return VerifyPaymentResult{}, fmt.Errorf("reference is required")
	}

	order, err := c.orders.FindByPaystackReference(ctx, reference)
	if err != nil {
		return VerifyPaymentResult{}, err
	}
	model.NormalizeOrder(order)

	out := VerifyPaymentResult{
		OrderID:        order.ID.Hex(),
		Status:         order.Status,
		TrackingNumber: order.TrackingNumber,
		CustomerEmail:  order.Customer.Email,
	}

	if c.paystack == nil {
		return out, nil
	}

	v, err := c.paystack.VerifyTransaction(ctx, reference)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("verify paystack: %w", err)
	}
	out.PaystackStatus = v.Status

	// Reconcile when Paystack confirms success but webhook has not landed yet.
	if strings.EqualFold(v.Status, "success") {
		switch order.Status {
		case model.OrderStatusPendingPayment, model.OrderStatusAbandoned:
			if v.Amount > 0 && v.Amount != order.TotalAmountKobo {
				return VerifyPaymentResult{}, fmt.Errorf("payment amount mismatch")
			}
			if err := c.HandleChargeSuccess(ctx, reference, v.Amount); err != nil {
				return VerifyPaymentResult{}, err
			}
			order, err = c.orders.FindByPaystackReference(ctx, reference)
			if err != nil {
				return VerifyPaymentResult{}, err
			}
			model.NormalizeOrder(order)
			out.Status = order.Status
			out.TrackingNumber = order.TrackingNumber
		}
	}

	return out, nil
}

type AbandonPaymentInput struct {
	Reference string
	Email     string
}

func (c *PaymentController) Abandon(ctx context.Context, in AbandonPaymentInput) error {
	reference := strings.TrimSpace(in.Reference)
	email := strings.TrimSpace(in.Email)
	if reference == "" {
		return fmt.Errorf("reference is required")
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}

	order, err := c.orders.FindByPaystackReference(ctx, reference)
	if err != nil {
		return err
	}
	model.NormalizeOrder(order)

	if order.Status == model.OrderStatusAbandoned {
		return nil
	}

	if !order.MayAbandon() {
		return ErrAbandonConflict
	}

	if !strings.EqualFold(strings.TrimSpace(order.Customer.Email), email) {
		return ErrAbandonEmailMismatch
	}

	now := time.Now().UTC()
	order.Status = model.OrderStatusAbandoned
	order.AbandonedAt = &now
	order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
		Status: model.OrderStatusAbandoned,
		At:     now,
		Note:   "Payment cancelled by customer",
	})

	return c.orders.Update(ctx, order)
}

func (c *PaymentController) HandleChargeSuccess(ctx context.Context, reference string, amountKobo int64) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return fmt.Errorf("reference is required")
	}

	order, err := c.orders.FindByPaystackReference(ctx, reference)
	if err != nil {
		return err
	}
	model.NormalizeOrder(order)

	if amountKobo > 0 && amountKobo != order.TotalAmountKobo {
		return fmt.Errorf("payment amount mismatch")
	}

	now := time.Now().UTC()
	statusChanged := markOrderPaidIfUnpaid(order, now)

	if err := c.orders.Update(ctx, order); err != nil {
		return err
	}

	if statusChanged {
		c.log.Info("order marked paid from webhook", "orderId", order.ID.Hex(), "reference", reference)
	}

	return c.maybeSendPaymentEmails(ctx, order)
}

// markOrderPaidIfUnpaid transitions pending_payment or abandoned orders to paid.
// Returns true when status changed to paid.
func markOrderPaidIfUnpaid(order *model.Order, now time.Time) bool {
	switch order.Status {
	case model.OrderStatusPendingPayment, model.OrderStatusAbandoned:
		order.Status = model.OrderStatusPaid
		order.AbandonedAt = nil
		order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
			Status: model.OrderStatusPaid,
			At:     now,
			Note:   "Payment confirmed",
		})
		if order.PaidAt == nil {
			order.PaidAt = &now
		}
		if order.TrackingNumber == "" {
			order.TrackingNumber = model.NewTrackingNumber(now)
		}
		return true
	default:
		if order.PaidAt == nil {
			order.PaidAt = &now
		}
		if order.TrackingNumber == "" {
			order.TrackingNumber = model.NewTrackingNumber(now)
		}
		return false
	}
}

func (c *PaymentController) maybeSendPaymentEmails(ctx context.Context, order *model.Order) error {
	if order.Notifications.EmailedAt != nil {
		return nil
	}
	if c.mail == nil || c.adminEmail == "" {
		c.log.Info("skipping payment emails: smtp not configured")
		now := time.Now().UTC()
		order.Notifications.EmailedAt = &now
		return c.orders.Update(ctx, order)
	}

	trackURL := c.clientPublicURL + "/track"
	if c.clientPublicURL == "" {
		trackURL = "/track"
	}

	custSubject, custHTML, custPlain, err := mail.OrderCustomerConfirmedEmail(order, trackURL)
	if err != nil {
		return fmt.Errorf("render customer email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(order.Customer.Email, custSubject, custPlain, custHTML); err != nil {
		return fmt.Errorf("customer email: %w", err)
	}

	adminSubject, adminHTML, adminPlain, err := mail.OrderAdminPaidEmail(order)
	if err != nil {
		return fmt.Errorf("render admin email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(c.adminEmail, adminSubject, adminPlain, adminHTML); err != nil {
		return fmt.Errorf("admin email: %w", err)
	}

	now := time.Now().UTC()
	order.Notifications.EmailedAt = &now
	return c.orders.Update(ctx, order)
}

type TrackOrderInput struct {
	TrackingNumber string
	Email          string
}

type TrackOrderView struct {
	TrackingNumber  string                     `json:"trackingNumber"`
	Status          model.OrderStatus          `json:"status"`
	StatusHistory   []model.StatusHistoryEntry `json:"statusHistory"`
	TotalAmountKobo int64                      `json:"totalAmountKobo"`
	Items           []model.OrderItem          `json:"items"`
}

func (c *PaymentController) Track(ctx context.Context, in TrackOrderInput) (TrackOrderView, error) {
	tracking := strings.TrimSpace(in.TrackingNumber)
	email := strings.TrimSpace(in.Email)
	if tracking == "" || email == "" {
		return TrackOrderView{}, fmt.Errorf("trackingNumber and email are required")
	}

	order, err := c.orders.FindForTrack(ctx, tracking, email)
	if err != nil {
		return TrackOrderView{}, err
	}
	model.NormalizeOrder(order)

	return TrackOrderView{
		TrackingNumber:  order.TrackingNumber,
		Status:          order.Status,
		StatusHistory:   order.StatusHistory,
		TotalAmountKobo: order.TotalAmountKobo,
		Items:           order.Items,
	}, nil
}

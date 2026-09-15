package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
)

var (
	ErrCustomOrderConflict   = errors.New("order cannot be accepted or rejected in its current state")
	ErrNotCustomOrder        = errors.New("order is not a custom order")
	ErrPaystackNotConfigured = errors.New("paystack is not configured")
)

func IsCustomOrderConflict(err error) bool {
	return errors.Is(err, ErrCustomOrderConflict)
}

func IsNotCustomOrder(err error) bool {
	return errors.Is(err, ErrNotCustomOrder)
}

type AcceptCustomOrderInput struct {
	AmountKobo int64
}

type RejectCustomOrderInput struct {
	Reason string
}

type AcceptCustomOrderResult struct {
	Order              *model.Order `json:"order"`
	AuthorizationURL   string       `json:"authorizationUrl"`
	PaystackReference  string       `json:"paystackReference"`
}

type ResendPaymentLinkResult struct {
	Order             *model.Order `json:"order"`
	AuthorizationURL  string       `json:"authorizationUrl"`
	PaystackReference string       `json:"paystackReference"`
}

// AcceptCustom sets the agreed amount, moves created → pending_payment, and emails a Paystack payment link.
func (c *OrderController) AcceptCustom(ctx context.Context, id primitive.ObjectID, in AcceptCustomOrderInput) (AcceptCustomOrderResult, error) {
	if in.AmountKobo < 1 {
		return AcceptCustomOrderResult{}, fmt.Errorf("amountKobo must be at least 1")
	}
	if c.paystack == nil {
		return AcceptCustomOrderResult{}, ErrPaystackNotConfigured
	}

	order, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return AcceptCustomOrderResult{}, err
	}
	model.NormalizeOrder(order)

	if order.OrderType != model.OrderTypeCustom {
		return AcceptCustomOrderResult{}, ErrNotCustomOrder
	}
	if !model.AllowedAdminTransition(order.OrderType, order.Status, model.OrderStatusPendingPayment) {
		return AcceptCustomOrderResult{}, ErrCustomOrderConflict
	}

	ref := strings.TrimSpace(order.PaystackReference)
	if ref == "" {
		ref = newPaystackReference()
	}

	init, err := c.paystack.InitializeTransaction(ctx, order.Customer.Email, in.AmountKobo, ref)
	if err != nil {
		return AcceptCustomOrderResult{}, fmt.Errorf("initialize paystack: %w", err)
	}
	if init.AuthorizationURL == "" {
		return AcceptCustomOrderResult{}, fmt.Errorf("paystack did not return a payment link")
	}
	if init.Reference != "" {
		ref = init.Reference
	}

	now := time.Now().UTC()
	order.TotalAmountKobo = in.AmountKobo
	order.PaystackReference = ref
	order.Status = model.OrderStatusPendingPayment
	order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
		Status: model.OrderStatusPendingPayment,
		At:     now,
		Note:   fmt.Sprintf("Accepted; agreed amount %s", mail.FormatNGN(in.AmountKobo)),
	})

	if err := c.orders.Update(ctx, order); err != nil {
		return AcceptCustomOrderResult{}, fmt.Errorf("update order: %w", err)
	}

	emailOrder := *order
	paymentURL := init.AuthorizationURL
	go func() {
		c.log.Info("custom order accept email sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
		if err := c.sendCustomPaymentLinkEmail(&emailOrder, paymentURL, false); err != nil {
			c.log.Error("custom order accept email failed",
				"orderId", emailOrder.ID.Hex(),
				"err", err,
			)
			return
		}
		c.log.Info("custom order accept email sent", "orderId", emailOrder.ID.Hex())
	}()

	return AcceptCustomOrderResult{
		Order:             order,
		AuthorizationURL:  init.AuthorizationURL,
		PaystackReference: ref,
	}, nil
}

// RejectCustom moves created → rejected with a required reason and emails the customer.
func (c *OrderController) RejectCustom(ctx context.Context, id primitive.ObjectID, in RejectCustomOrderInput) (*model.Order, error) {
	reason := strings.TrimSpace(in.Reason)
	order, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	model.NormalizeOrder(order)

	if order.OrderType != model.OrderTypeCustom {
		return nil, ErrNotCustomOrder
	}
	if !model.AllowedAdminTransition(order.OrderType, order.Status, model.OrderStatusRejected) {
		return nil, ErrCustomOrderConflict
	}
	if err := model.ValidateStatusChangeNote(order.OrderType, order.Status, model.OrderStatusRejected, reason); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	order.Status = model.OrderStatusRejected
	order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
		Status: model.OrderStatusRejected,
		At:     now,
		Note:   reason,
	})

	if err := c.orders.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}

	emailOrder := *order
	go func() {
		c.log.Info("custom order reject email sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
		if err := c.sendStatusUpdateEmail(&emailOrder, reason); err != nil {
			c.log.Error("custom order reject email failed",
				"orderId", emailOrder.ID.Hex(),
				"err", err,
			)
			return
		}
		c.log.Info("custom order reject email sent", "orderId", emailOrder.ID.Hex())
	}()

	return order, nil
}

// ResendCustomPaymentLink mints a fresh Paystack checkout link for a custom order awaiting payment.
func (c *OrderController) ResendCustomPaymentLink(ctx context.Context, id primitive.ObjectID) (ResendPaymentLinkResult, error) {
	if c.paystack == nil {
		return ResendPaymentLinkResult{}, ErrPaystackNotConfigured
	}

	order, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return ResendPaymentLinkResult{}, err
	}
	model.NormalizeOrder(order)

	if order.OrderType != model.OrderTypeCustom {
		return ResendPaymentLinkResult{}, ErrNotCustomOrder
	}
	if order.Status != model.OrderStatusPendingPayment {
		return ResendPaymentLinkResult{}, ErrCustomOrderConflict
	}
	if order.TotalAmountKobo < 1 {
		return ResendPaymentLinkResult{}, fmt.Errorf("order total amount is invalid")
	}

	ref := newPaystackReference()
	init, err := c.paystack.InitializeTransaction(ctx, order.Customer.Email, order.TotalAmountKobo, ref)
	if err != nil {
		return ResendPaymentLinkResult{}, fmt.Errorf("initialize paystack: %w", err)
	}
	if init.AuthorizationURL == "" {
		return ResendPaymentLinkResult{}, fmt.Errorf("paystack did not return a payment link")
	}
	if init.Reference != "" {
		ref = init.Reference
	}

	now := time.Now().UTC()
	order.PaystackReference = ref
	order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
		Status: model.OrderStatusPendingPayment,
		At:     now,
		Note:   "Payment link resent",
	})

	if err := c.orders.Update(ctx, order); err != nil {
		return ResendPaymentLinkResult{}, fmt.Errorf("update order: %w", err)
	}

	emailOrder := *order
	paymentURL := init.AuthorizationURL
	go func() {
		c.log.Info("custom order payment link resend email sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
		if err := c.sendCustomPaymentLinkEmail(&emailOrder, paymentURL, true); err != nil {
			c.log.Error("custom order payment link resend email failed",
				"orderId", emailOrder.ID.Hex(),
				"err", err,
			)
			return
		}
		c.log.Info("custom order payment link resend email sent", "orderId", emailOrder.ID.Hex())
	}()

	return ResendPaymentLinkResult{
		Order:             order,
		AuthorizationURL:  init.AuthorizationURL,
		PaystackReference: ref,
	}, nil
}

func (c *OrderController) sendCustomPaymentLinkEmail(order *model.Order, paymentURL string, resent bool) error {
	if c.mail == nil {
		c.log.Info("skipping custom payment link email: smtp not configured", "orderId", order.ID.Hex())
		return nil
	}

	trackURL := ""
	if c.clientPublicURL != "" {
		trackURL = c.clientPublicURL + "/track"
	}

	subject, html, plain, err := mail.OrderCustomPaymentLinkEmail(order, paymentURL, trackURL, resent)
	if err != nil {
		return fmt.Errorf("render payment link email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(order.Customer.Email, subject, plain, html); err != nil {
		return fmt.Errorf("send payment link email: %w", err)
	}
	return nil
}

package controller

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/paystack"
)

func TestAcceptCustom_requiresPositiveAmount(t *testing.T) {
	ctrl := NewOrderController(nil, nil, nil, &stubPaymentLinker{}, nil, slog.Default(), "")
	_, err := ctrl.AcceptCustom(context.Background(), primitive.NewObjectID(), AcceptCustomOrderInput{AmountKobo: 0})
	if err == nil || !strings.Contains(err.Error(), "amountKobo") {
		t.Fatalf("got %v", err)
	}
}

func TestAcceptCustom_requiresPaystack(t *testing.T) {
	ctrl := NewOrderController(nil, nil, nil, nil, nil, slog.Default(), "")
	_, err := ctrl.AcceptCustom(context.Background(), primitive.NewObjectID(), AcceptCustomOrderInput{AmountKobo: 1000})
	if !errors.Is(err, ErrPaystackNotConfigured) {
		t.Fatalf("got %v", err)
	}
}

func TestResendCustomPaymentLink_requiresPaystack(t *testing.T) {
	ctrl := NewOrderController(nil, nil, nil, nil, nil, slog.Default(), "")
	_, err := ctrl.ResendCustomPaymentLink(context.Background(), primitive.NewObjectID())
	if !errors.Is(err, ErrPaystackNotConfigured) {
		t.Fatalf("got %v", err)
	}
}

func TestCustomAcceptTransitionHelpers(t *testing.T) {
	if !model.AllowedAdminTransition(model.OrderTypeCustom, model.OrderStatusCreated, model.OrderStatusPendingPayment) {
		t.Fatal("custom accept transition missing")
	}
	if !model.AllowedAdminTransition(model.OrderTypeCustom, model.OrderStatusCreated, model.OrderStatusRejected) {
		t.Fatal("custom reject transition missing")
	}
	if model.AllowedAdminTransition(model.OrderTypeShop, model.OrderStatusCreated, model.OrderStatusPendingPayment) {
		t.Fatal("shop must not accept-via-created")
	}
	if err := model.ValidateStatusChangeNote(model.OrderTypeCustom, model.OrderStatusCreated, model.OrderStatusRejected, ""); err == nil {
		t.Fatal("reject without reason should fail")
	}
	if !model.AllowedAdminTransition(model.OrderTypeCustom, model.OrderStatusPendingPayment, model.OrderStatusPaid) {
		t.Fatal("custom manual paid transition missing")
	}
}

type stubPaymentLinker struct {
	result  paystack.InitializeResult
	err     error
	calls   int
	lastRef string
	lastAmt int64
}

func (s *stubPaymentLinker) InitializeTransaction(_ context.Context, email string, amountKobo int64, reference string) (paystack.InitializeResult, error) {
	s.calls++
	s.lastRef = reference
	s.lastAmt = amountKobo
	if s.err != nil {
		return paystack.InitializeResult{}, s.err
	}
	out := s.result
	if out.Reference == "" {
		out.Reference = reference
	}
	if out.AuthorizationURL == "" {
		out.AuthorizationURL = "https://checkout.paystack.com/test"
	}
	if out.AccessCode == "" {
		out.AccessCode = "access_test"
	}
	_ = email
	return out, nil
}

package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/paystack"
)

type PaymentHandler struct {
	ctrl      *controller.PaymentController
	secretKey string
}

func NewPaymentHandler(ctrl *controller.PaymentController, paystackSecret string) *PaymentHandler {
	return &PaymentHandler{ctrl: ctrl, secretKey: paystackSecret}
}

type initializePaymentRequest struct {
	OrderID string `json:"orderId"`
}

func (h *PaymentHandler) Initialize(c *gin.Context) {
	var req initializePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.OrderID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid orderId"})
		return
	}

	out, err := h.ctrl.Initialize(c.Request.Context(), id)
	if err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *PaymentHandler) Verify(c *gin.Context) {
	ref := strings.TrimSpace(c.Query("reference"))
	out, err := h.ctrl.Verify(c.Request.Context(), ref)
	if err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *PaymentHandler) PaystackWebhook(c *gin.Context) {
	raw, ok := rawBodyFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing raw body"})
		return
	}

	sig := c.GetHeader("x-paystack-signature")
	if !paystack.VerifyWebhookSignature(h.secretKey, raw, sig) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var evt paystack.WebhookEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	if evt.Event != "charge.success" {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	data, err := paystack.ParseChargeSuccess(evt.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ctrl.HandleChargeSuccess(c.Request.Context(), data.Reference, data.Amount); err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

type trackOrderRequest struct {
	TrackingNumber string `json:"trackingNumber"`
	Email          string `json:"email"`
}

func (h *PaymentHandler) TrackOrder(c *gin.Context) {
	var req trackOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	out, err := h.ctrl.Track(c.Request.Context(), controller.TrackOrderInput{
		TrackingNumber: req.TrackingNumber,
		Email:          req.Email,
	})
	if err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

type abandonPaymentRequest struct {
	Reference string `json:"reference"`
	Email     string `json:"email"`
}

func (h *PaymentHandler) Abandon(c *gin.Context) {
	var req abandonPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := h.ctrl.Abandon(c.Request.Context(), controller.AbandonPaymentInput{
		Reference: req.Reference,
		Email:     req.Email,
	})
	if err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func writePaymentError(c *gin.Context, err error) {
	if controller.IsOrderNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if controller.IsAbandonEmailMismatch(err) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if controller.IsAbandonConflict(err) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "not configured") {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": msg})
		return
	}
	if strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "mismatch") ||
		strings.Contains(msg, "not awaiting") {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "payment request failed"})
}

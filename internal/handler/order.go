package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pagination"
	"homeessentials/backend/internal/pricing"
	"homeessentials/backend/internal/repository"
)

type OrderHandler struct {
	ctrl *controller.OrderController
}

func NewOrderHandler(ctrl *controller.OrderController) *OrderHandler {
	return &OrderHandler{ctrl: ctrl}
}

type orderLineDTO struct {
	ProductID string `json:"productId"`
	Variant   string `json:"variant"`
	Size      string `json:"size"`
	Unit      string `json:"unit"`
	Quantity  int    `json:"quantity"`
}

type customerDTO struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	DeliveryAddress string `json:"deliveryAddress"`
}

type createOrderRequest struct {
	Items    []orderLineDTO `json:"items"`
	Customer customerDTO    `json:"customer"`
}

func (h *OrderHandler) CreatePublic(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	in, err := createRequestToInput(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.ctrl.Create(c.Request.Context(), in)
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	order, err := h.ctrl.Get(c.Request.Context(), id)
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) ListAdmin(c *gin.Context) {
	f, err := parseOrderListFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.ctrl.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, out)
}

func parseOrderListFilter(c *gin.Context) (repository.OrderListFilter, error) {
	var f repository.OrderListFilter

	pg, err := pagination.ParseQuery(c.Query("page"), c.Query("page_size"))
	if err != nil {
		return f, err
	}
	f.Page = pg.Page
	f.PageSize = pg.PageSize

	if statusStr := c.Query("status"); statusStr != "" {
		status, err := model.ParseOrderListFilterStatus(statusStr)
		if err != nil {
			return f, err
		}
		f.Status = &status
	}

	if typeStr := strings.TrimSpace(c.Query("orderType")); typeStr != "" {
		orderType, err := model.ParseOrderType(typeStr)
		if err != nil {
			return f, err
		}
		f.OrderType = &orderType
	}

	return f, nil
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *OrderHandler) UpdateStatusAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var req updateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	status, err := model.ParseOrderStatus(req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.ctrl.UpdateStatus(c.Request.Context(), id, controller.UpdateOrderStatusInput{
		Status: status,
		Note:   req.Note,
	})
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) DeleteAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	if err := h.ctrl.DeleteAbandoned(c.Request.Context(), id); err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func createRequestToInput(req createOrderRequest) (controller.CreateOrderInput, error) {
	lines := make([]controller.OrderLineInput, 0, len(req.Items))
	for i, line := range req.Items {
		pid, err := primitive.ObjectIDFromHex(strings.TrimSpace(line.ProductID))
		if err != nil {
			return controller.CreateOrderInput{}, errors.New("invalid productId on line " + strconv.Itoa(i+1))
		}
		unit, err := model.ParseOrderUnit(line.Unit)
		if err != nil {
			return controller.CreateOrderInput{}, err
		}
		lines = append(lines, controller.OrderLineInput{
			ProductID: pid,
			Variant:   line.Variant,
			Size:      line.Size,
			Unit:      unit,
			Quantity:  line.Quantity,
		})
	}

	return controller.CreateOrderInput{
		Items: lines,
		Customer: controller.CustomerInput{
			Name:            req.Customer.Name,
			Email:           req.Customer.Email,
			Phone:           req.Customer.Phone,
			DeliveryAddress: req.Customer.DeliveryAddress,
		},
	}, nil
}

func writeOrderError(c *gin.Context, err error) {
	if controller.IsOrderNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if controller.IsOrderDeleteNotAbandoned(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if errors.Is(err, pricing.ErrNoPriceForUnit) || errors.Is(err, pricing.ErrInvalidPricing) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := err.Error()
	if strings.Contains(msg, "product not found") ||
		strings.Contains(msg, "not available") ||
		strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "cannot change status") ||
		strings.Contains(msg, "already") ||
		strings.Contains(msg, "quantity") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "at least") {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	slog.Error("order request failed", "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "order request failed"})
}

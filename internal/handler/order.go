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
	Size      string `json:"size"`
	Color     string `json:"color"`
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

const maxCustomOrderUploadBytes = 5 << 20

func (h *OrderHandler) CreateCustomPublic(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxCustomOrderUploadBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	in, err := customOrderFormToInput(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, header, err := c.Request.FormFile("sampleImage")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sampleImage file is required"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	order, err := h.ctrl.CreateCustom(c.Request.Context(), in, controller.CustomSampleImage{
		Filename:    header.Filename,
		ContentType: contentType,
		Reader:      file,
	})
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusCreated, order)
}

func customOrderFormToInput(c *gin.Context) (controller.CreateCustomOrderInput, error) {
	qtyRaw := strings.TrimSpace(c.PostForm("quantity"))
	qty, err := strconv.Atoi(qtyRaw)
	if err != nil {
		return controller.CreateCustomOrderInput{}, errors.New("quantity must be a number")
	}
	offeredRaw := strings.TrimSpace(c.PostForm("offeredTotalKobo"))
	offered, err := strconv.ParseInt(offeredRaw, 10, 64)
	if err != nil {
		return controller.CreateCustomOrderInput{}, errors.New("offeredTotalKobo must be a number")
	}

	sizes, err := parseSizeCodesCSV(c.PostForm("sizes"))
	if err != nil {
		return controller.CreateCustomOrderInput{}, err
	}
	colors := parseCSVTokens(c.PostForm("colors"))

	return controller.CreateCustomOrderInput{
		Customer: controller.CustomerInput{
			Name:            c.PostForm("name"),
			Email:           c.PostForm("email"),
			Phone:           c.PostForm("phone"),
			DeliveryAddress: c.PostForm("deliveryAddress"),
		},
		Custom: controller.CustomRequestInput{
			Title:            c.PostForm("title"),
			Description:      c.PostForm("description"),
			Sizes:            sizes,
			Colors:           colors,
			Quantity:         qty,
			OfferedTotalKobo: offered,
		},
	}, nil
}

func parseSizeCodesCSV(raw string) ([]model.SizeCode, error) {
	parts := parseCSVTokens(raw)
	if len(parts) == 0 {
		return nil, errors.New("at least one size is required")
	}
	out := make([]model.SizeCode, 0, len(parts))
	for _, p := range parts {
		code, err := model.ParseSizeCode(p)
		if err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, nil
}

func parseCSVTokens(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

type acceptCustomOrderRequest struct {
	AmountKobo int64 `json:"amountKobo"`
}

func (h *OrderHandler) AcceptCustomAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var req acceptCustomOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	out, err := h.ctrl.AcceptCustom(c.Request.Context(), id, controller.AcceptCustomOrderInput{
		AmountKobo: req.AmountKobo,
	})
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

type rejectCustomOrderRequest struct {
	Reason string `json:"reason"`
}

func (h *OrderHandler) RejectCustomAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var req rejectCustomOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	order, err := h.ctrl.RejectCustom(c.Request.Context(), id, controller.RejectCustomOrderInput{
		Reason: req.Reason,
	})
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) ResendPaymentLinkAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	out, err := h.ctrl.ResendCustomPaymentLink(c.Request.Context(), id)
	if err != nil {
		writeOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
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
		size, err := model.ParseSizeCode(line.Size)
		if err != nil {
			return controller.CreateOrderInput{}, err
		}
		lines = append(lines, controller.OrderLineInput{
			ProductID: pid,
			Size:      size,
			Color:     line.Color,
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
	if controller.IsNotCustomOrder(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if controller.IsCustomOrderConflict(err) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, controller.ErrUploadNotConfigured) || errors.Is(err, controller.ErrPaystackNotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, controller.ErrInvalidImageType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image type"})
		return
	}

	if errors.Is(err, pricing.ErrNoTierForQty) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := err.Error()
	if strings.Contains(msg, "upload sample image") {
		slog.Error("custom order sample upload failed", "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to upload sample image"})
		return
	}
	if strings.Contains(msg, "create custom order") {
		slog.Error("custom order create failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "order request failed"})
		return
	}
	if strings.Contains(msg, "product not found") ||
		strings.Contains(msg, "not available") ||
		strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "cannot change status") ||
		strings.Contains(msg, "already") ||
		strings.Contains(msg, "quantity") ||
		strings.Contains(msg, "offeredTotalKobo") ||
		strings.Contains(msg, "amountKobo") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "at least") ||
		strings.Contains(msg, "reason") ||
		strings.Contains(msg, "paystack") {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	slog.Error("order request failed", "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "order request failed"})
}

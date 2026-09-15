package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pagination"
	"homeessentials/backend/internal/paystack"
	"homeessentials/backend/internal/pricing"
	"homeessentials/backend/internal/repository"
)

var ErrOrderDeleteNotAbandoned = errors.New("only abandoned orders may be deleted")

func IsOrderDeleteNotAbandoned(err error) bool {
	return errors.Is(err, ErrOrderDeleteNotAbandoned)
}

type OrderLineInput struct {
	ProductID primitive.ObjectID
	Size      model.SizeCode
	Color     string
	Quantity  int
}

type CustomerInput struct {
	Name            string
	Email           string
	Phone           string
	DeliveryAddress string
}

type CreateOrderInput struct {
	Items    []OrderLineInput
	Customer CustomerInput
}

type UpdateOrderStatusInput struct {
	Status model.OrderStatus
	Note   string
}

type OrderController struct {
	orders          *repository.OrderRepository
	products        *repository.ProductRepository
	recreateImages  RecreateImageUploader
	paystack        PaymentLinker
	mail            *mail.Sender
	log             *slog.Logger
	clientPublicURL string
}

// PaymentLinker initializes Paystack hosted checkout sessions.
type PaymentLinker interface {
	InitializeTransaction(ctx context.Context, email string, amountKobo int64, reference string) (paystack.InitializeResult, error)
}

func NewOrderController(
	orders *repository.OrderRepository,
	products *repository.ProductRepository,
	recreateImages RecreateImageUploader,
	ps PaymentLinker,
	mailer *mail.Sender,
	log *slog.Logger,
	clientPublicURL string,
) *OrderController {
	return &OrderController{
		orders:          orders,
		products:        products,
		recreateImages:  recreateImages,
		paystack:        ps,
		mail:            mailer,
		log:             log,
		clientPublicURL: strings.TrimSuffix(clientPublicURL, "/"),
	}
}

func (c *OrderController) Create(ctx context.Context, in CreateOrderInput) (*model.Order, error) {
	if err := validateCustomer(in.Customer); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 {
		return nil, fmt.Errorf("at least one order line is required")
	}

	ids := make([]primitive.ObjectID, 0, len(in.Items))
	for _, line := range in.Items {
		if line.ProductID.IsZero() {
			return nil, fmt.Errorf("productId is required on each line")
		}
		if line.Quantity < 1 {
			return nil, fmt.Errorf("quantity must be at least 1")
		}
		if !line.Size.Valid() {
			return nil, fmt.Errorf("invalid size %q", line.Size)
		}
		if strings.TrimSpace(line.Color) == "" {
			return nil, fmt.Errorf("color is required on each line")
		}
		ids = append(ids, line.ProductID)
	}

	productsByID, err := c.products.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load products: %w", err)
	}

	items, total, err := buildOrderItems(productsByID, in.Items)
	if err != nil {
		return nil, err
	}

	placedAt := time.Now().UTC()
	order := &model.Order{
		OrderType:         model.OrderTypeShop,
		Items:             items,
		Customer:          normalizeCustomer(in.Customer),
		Status:            model.OrderStatusPendingPayment,
		StatusHistory:     []model.StatusHistoryEntry{{Status: model.OrderStatusPendingPayment, At: placedAt}},
		TotalAmountKobo:   total,
		PaystackReference: newPaystackReference(),
	}

	if err := c.orders.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return order, nil
}

func buildOrderItems(productsByID map[primitive.ObjectID]model.Product, lines []OrderLineInput) ([]model.OrderItem, int64, error) {
	items := make([]model.OrderItem, 0, len(lines))
	var total int64

	for i, line := range lines {
		p, ok := productsByID[line.ProductID]
		if !ok {
			return nil, 0, fmt.Errorf("line %d: product not found", i+1)
		}
		if !p.Active {
			return nil, 0, fmt.Errorf("line %d: product is not available", i+1)
		}

		color := strings.TrimSpace(line.Color)
		if !productHasColor(p.Colors, color) {
			return nil, 0, fmt.Errorf("line %d: color %q is not available for this product", i+1, color)
		}

		tiers, err := tiersForSize(p.Sizes, line.Size)
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: %w", i+1, err)
		}

		unit, err := pricing.UnitPriceForQty(tiers, line.Quantity)
		if err != nil {
			return nil, 0, fmt.Errorf("line %d: %w", i+1, err)
		}

		lineTotal := unit * int64(line.Quantity)
		items = append(items, model.OrderItem{
			ProductID:     p.ID,
			ProductTitle:  p.Title,
			Size:          line.Size,
			Color:         color,
			ImageURL:      p.ImageURLForColor(color),
			Quantity:      line.Quantity,
			UnitPriceKobo: unit,
			LineTotalKobo: lineTotal,
		})
		total += lineTotal
	}

	return items, total, nil
}

func (c *OrderController) Get(ctx context.Context, id primitive.ObjectID) (*model.Order, error) {
	o, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	model.NormalizeOrder(o)
	return o, nil
}

func (c *OrderController) List(ctx context.Context, f repository.OrderListFilter) (pagination.Paginated[model.Order], error) {
	orders, total, err := c.orders.List(ctx, f)
	if err != nil {
		return pagination.Paginated[model.Order]{}, err
	}
	for i := range orders {
		model.NormalizeOrder(&orders[i])
	}
	return pagination.NewPaginated(orders, total, f.Page, f.PageSize), nil
}

func (c *OrderController) UpdateStatus(ctx context.Context, id primitive.ObjectID, in UpdateOrderStatusInput) (*model.Order, error) {
	if !in.Status.AdminSettable() {
		return nil, fmt.Errorf("invalid order status")
	}

	order, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	model.NormalizeOrder(order)

	if !model.AllowedAdminTransition(order.OrderType, order.Status, in.Status) {
		return nil, fmt.Errorf("cannot change status from %s to %s", order.Status, in.Status)
	}

	previousStatus := order.Status
	note := strings.TrimSpace(in.Note)
	if err := model.ValidateStatusChangeNote(order.OrderType, order.Status, in.Status, note); err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	if in.Status == model.OrderStatusPaid {
		if order.PaidAt == nil {
			order.PaidAt = &now
		}
		if order.TrackingNumber == "" {
			order.TrackingNumber = model.NewTrackingNumber(now)
		}
	}

	order.Status = in.Status
	order.StatusHistory = append(order.StatusHistory, model.StatusHistoryEntry{
		Status: in.Status,
		At:     now,
		Note:   note,
	})

	if err := c.orders.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	emailOrder := *order
	go func() {
		c.log.Info("status update email sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
			"from", previousStatus,
			"to", in.Status,
		)
		if err := c.sendStatusUpdateEmail(&emailOrder, note); err != nil {
			c.log.Error("status update email failed",
				"orderId", emailOrder.ID.Hex(),
				"from", previousStatus,
				"to", in.Status,
				"err", err,
			)
			return
		}
		c.log.Info("status update email sent",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
			"to", in.Status,
		)
	}()

	return order, nil
}

func (c *OrderController) DeleteAbandoned(ctx context.Context, id primitive.ObjectID) error {
	order, err := c.orders.FindByID(ctx, id)
	if err != nil {
		return err
	}
	model.NormalizeOrder(order)
	if order.Status != model.OrderStatusAbandoned {
		return ErrOrderDeleteNotAbandoned
	}
	return c.orders.DeleteByID(ctx, id)
}

func (c *OrderController) sendStatusUpdateEmail(order *model.Order, note string) error {
	if c.mail == nil {
		c.log.Info("skipping status email: smtp not configured", "orderId", order.ID.Hex())
		return nil
	}

	trackURL := ""
	if c.clientPublicURL != "" {
		trackURL = c.clientPublicURL + "/track"
	}

	subject, html, plain, err := mail.OrderStatusUpdateEmail(order, note, trackURL)
	if err != nil {
		return fmt.Errorf("render status email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(order.Customer.Email, subject, plain, html); err != nil {
		return fmt.Errorf("send status email: %w", err)
	}
	return nil
}

func IsOrderNotFound(err error) bool {
	return errors.Is(err, repository.ErrOrderNotFound)
}

func validateCustomer(c CustomerInput) error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("customer name is required")
	}
	if strings.TrimSpace(c.Email) == "" {
		return fmt.Errorf("customer email is required")
	}
	if strings.TrimSpace(c.Phone) == "" {
		return fmt.Errorf("customer phone is required")
	}
	if strings.TrimSpace(c.DeliveryAddress) == "" {
		return fmt.Errorf("customer deliveryAddress is required")
	}
	return nil
}

func normalizeCustomer(c CustomerInput) model.CustomerInfo {
	return model.CustomerInfo{
		Name:            strings.TrimSpace(c.Name),
		Email:           strings.TrimSpace(c.Email),
		Phone:           strings.TrimSpace(c.Phone),
		DeliveryAddress: strings.TrimSpace(c.DeliveryAddress),
	}
}

func productHasColor(colors []string, want string) bool {
	for _, c := range colors {
		if strings.EqualFold(strings.TrimSpace(c), want) {
			return true
		}
	}
	return false
}

func tiersForSize(sizes []model.SizeVariant, code model.SizeCode) ([]model.QtyTier, error) {
	for _, sv := range sizes {
		if sv.Code == code {
			return sv.Tiers, nil
		}
	}
	return nil, fmt.Errorf("size %q is not available for this product", code)
}

func newPaystackReference() string {
	return "fol_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

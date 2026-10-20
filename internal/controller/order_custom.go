package controller

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/model"
)

// RecreateImageUploader stores sample images for custom (custom) orders.
type RecreateImageUploader interface {
	UploadRecreateImage(filename string, r io.Reader, contentType string) (string, error)
}

type CustomRequestInput struct {
	Title    string
	Quantity int
}

type CreateCustomOrderInput struct {
	Customer CustomerInput
	Custom   CustomRequestInput
}

type CustomSampleImage struct {
	Filename    string
	ContentType string
	Reader      io.Reader
}

func (c *OrderController) CreateCustom(ctx context.Context, in CreateCustomOrderInput, sample CustomSampleImage) (*model.Order, error) {
	if err := validateCustomer(in.Customer); err != nil {
		return nil, err
	}
	custom, err := validateCustomRequest(in.Custom)
	if err != nil {
		return nil, err
	}
	if sample.Reader == nil {
		return nil, fmt.Errorf("sample image is required")
	}
	if c.recreateImages == nil {
		return nil, ErrUploadNotConfigured
	}

	contentType := strings.TrimSpace(strings.ToLower(sample.ContentType))
	if !isAllowedImageType(contentType) {
		contentType = imageContentTypeFromFilename(sample.Filename)
	}
	if !isAllowedImageType(contentType) {
		return nil, ErrInvalidImageType
	}

	imageURL, err := c.recreateImages.UploadRecreateImage(sample.Filename, sample.Reader, contentType)
	if err != nil {
		return nil, fmt.Errorf("upload sample image: %w", err)
	}
	custom.SampleImageURL = imageURL

	placedAt := time.Now().UTC()
	order := &model.Order{
		OrderType: model.OrderTypeCustom,
		Items:     []model.OrderItem{},
		Custom:    custom,
		Customer:  normalizeCustomer(in.Customer),
		Status:    model.OrderStatusCreated,
		StatusHistory: []model.StatusHistoryEntry{{
			Status: model.OrderStatusCreated,
			At:     placedAt,
			Note:   "Custom mat request submitted",
		}},
		SubtotalAmountKobo: 0,
		DeliveryFeeKobo:    0,
		TotalAmountKobo:    0,
		PaystackReference:  newPaystackReference(),
		TrackingNumber:     model.NewTrackingNumber(placedAt),
	}

	if err := c.orders.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create custom order: %w", err)
	}

	emailOrder := *order
	go func() {
		c.log.Info("custom order acknowledgement emails sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
		if err := c.sendCustomOrderCreatedEmails(&emailOrder); err != nil {
			c.log.Error("custom order acknowledgement emails failed",
				"orderId", emailOrder.ID.Hex(),
				"err", err,
			)
			return
		}
		c.log.Info("custom order acknowledgement emails sent",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
	}()

	return order, nil
}

func validateCustomRequest(in CustomRequestInput) (*model.CustomRequest, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if in.Quantity < 1 {
		return nil, fmt.Errorf("quantity must be at least 1")
	}

	return &model.CustomRequest{
		Title:    title,
		Quantity: in.Quantity,
	}, nil
}

func (c *OrderController) sendCustomOrderCreatedEmails(order *model.Order) error {
	if c.mail == nil {
		c.log.Info("skipping custom order emails: smtp not configured", "orderId", order.ID.Hex())
		return nil
	}

	trackURL := ""
	if c.clientPublicURL != "" {
		trackURL = c.clientPublicURL + "/track"
	}

	subject, html, plain, err := mail.OrderCustomCreatedEmail(order, trackURL)
	if err != nil {
		return fmt.Errorf("render custom order email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(order.Customer.Email, subject, plain, html); err != nil {
		return fmt.Errorf("send custom order email: %w", err)
	}

	if c.adminEmail == "" {
		c.log.Info("skipping custom order admin email: admin notify not configured", "orderId", order.ID.Hex())
		return nil
	}
	adminSubject, adminHTML, adminPlain, err := mail.OrderCustomAdminNotifyEmail(order)
	if err != nil {
		return fmt.Errorf("render custom order admin email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(c.adminEmail, adminSubject, adminPlain, adminHTML); err != nil {
		return fmt.Errorf("send custom order admin email: %w", err)
	}
	return nil
}

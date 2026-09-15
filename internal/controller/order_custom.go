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

// RecreateImageUploader stores sample images for custom (recreate) orders.
type RecreateImageUploader interface {
	UploadRecreateImage(filename string, r io.Reader, contentType string) (string, error)
}

type CustomRequestInput struct {
	Title            string
	Description      string
	Sizes            []model.SizeCode
	Colors           []string
	Quantity         int
	OfferedTotalKobo int64
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
			Note:   "Custom order request submitted",
		}},
		// Offered amount until admin accept sets the agreed charge.
		TotalAmountKobo:   custom.OfferedTotalKobo,
		PaystackReference: newPaystackReference(),
		TrackingNumber:    model.NewTrackingNumber(placedAt),
	}

	if err := c.orders.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create custom order: %w", err)
	}

	emailOrder := *order
	go func() {
		c.log.Info("custom order acknowledgement email sending",
			"orderId", emailOrder.ID.Hex(),
			"customerEmail", emailOrder.Customer.Email,
		)
		if err := c.sendCustomOrderCreatedEmail(&emailOrder); err != nil {
			c.log.Error("custom order acknowledgement email failed",
				"orderId", emailOrder.ID.Hex(),
				"err", err,
			)
			return
		}
		c.log.Info("custom order acknowledgement email sent",
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
	description := strings.TrimSpace(in.Description)
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if in.Quantity < 1 {
		return nil, fmt.Errorf("quantity must be at least 1")
	}
	if in.OfferedTotalKobo < 1 {
		return nil, fmt.Errorf("offeredTotalKobo must be at least 1")
	}
	if len(in.Sizes) == 0 {
		return nil, fmt.Errorf("at least one size is required")
	}

	sizes := make([]model.SizeCode, 0, len(in.Sizes))
	seenSize := make(map[model.SizeCode]struct{}, len(in.Sizes))
	for _, s := range in.Sizes {
		if !s.Valid() {
			return nil, fmt.Errorf("invalid size %q", s)
		}
		if _, ok := seenSize[s]; ok {
			continue
		}
		seenSize[s] = struct{}{}
		sizes = append(sizes, s)
	}

	colors := make([]string, 0, len(in.Colors))
	seenColor := make(map[string]struct{}, len(in.Colors))
	for _, raw := range in.Colors {
		color := strings.TrimSpace(raw)
		if color == "" {
			continue
		}
		key := strings.ToLower(color)
		if _, ok := seenColor[key]; ok {
			continue
		}
		seenColor[key] = struct{}{}
		colors = append(colors, color)
	}
	if len(colors) == 0 {
		return nil, fmt.Errorf("at least one color is required")
	}

	return &model.CustomRequest{
		Title:            title,
		Description:      description,
		Sizes:            sizes,
		Colors:           colors,
		Quantity:         in.Quantity,
		OfferedTotalKobo: in.OfferedTotalKobo,
	}, nil
}

func (c *OrderController) sendCustomOrderCreatedEmail(order *model.Order) error {
	if c.mail == nil {
		c.log.Info("skipping custom order email: smtp not configured", "orderId", order.ID.Hex())
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
	return nil
}

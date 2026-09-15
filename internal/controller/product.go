package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pagination"
	"homeessentials/backend/internal/pricing"
	"homeessentials/backend/internal/repository"
)

// ProductImageDeleter removes orphaned product images from object storage (optional).
type ProductImageDeleter interface {
	DeleteByPublicURL(publicURL string) error
}

type ProductInput struct {
	Title       string
	Description string
	Category    model.Category
	Colors      []string
	ColorImages []model.ColorImage
	Active      bool
	Sizes       []model.SizeVariant
}

var (
	ErrProductDeletePendingOrder    = errors.New("There is a pending order for this item.")
	ErrProductDeleteIncompleteOrder = errors.New("There is an incomplete order for this item.")
)

type ProductController struct {
	repo   *repository.ProductRepository
	orders *repository.OrderRepository
	images ProductImageDeleter
	log    *slog.Logger
}

func NewProductController(repo *repository.ProductRepository, orders *repository.OrderRepository, images ProductImageDeleter, log *slog.Logger) *ProductController {
	if log == nil {
		log = slog.Default()
	}
	return &ProductController{repo: repo, orders: orders, images: images, log: log}
}

func (c *ProductController) Create(ctx context.Context, in ProductInput) (*model.Product, error) {
	if err := validateProductInput(in); err != nil {
		return nil, err
	}

	p := &model.Product{
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		Category:    in.Category,
		Colors:      normalizeStrings(in.Colors),
		ColorImages: normalizeColorImages(in.ColorImages),
		Active:      in.Active,
		Sizes:       in.Sizes,
	}

	if err := c.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (c *ProductController) Update(ctx context.Context, id primitive.ObjectID, in ProductInput) (*model.Product, error) {
	if err := validateProductInput(in); err != nil {
		return nil, err
	}

	existing, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	previousURLs := productColorImageURLs(*existing)

	existing.Title = strings.TrimSpace(in.Title)
	existing.Description = strings.TrimSpace(in.Description)
	existing.Category = in.Category
	existing.Colors = normalizeStrings(in.Colors)
	existing.ColorImages = normalizeColorImages(in.ColorImages)
	existing.Active = in.Active
	existing.Sizes = in.Sizes

	if err := c.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	c.deleteOrphanedProductImages(previousURLs, productColorImageURLs(*existing))

	return existing, nil
}

func (c *ProductController) Get(ctx context.Context, id primitive.ObjectID) (*model.Product, error) {
	return c.repo.FindByID(ctx, id)
}

func (c *ProductController) List(ctx context.Context, f repository.ListFilter) (pagination.Paginated[model.Product], error) {
	products, total, err := c.repo.List(ctx, f)
	if err != nil {
		return pagination.Paginated[model.Product]{}, err
	}
	return pagination.NewPaginated(products, total, f.Page, f.PageSize), nil
}

func (c *ProductController) Delete(ctx context.Context, id primitive.ObjectID) error {
	if c.orders != nil {
		blockPending, blockIncomplete, err := c.orders.ProductDeleteBlockReason(ctx, id)
		if err != nil {
			return fmt.Errorf("check orders for product delete: %w", err)
		}
		if blockPending {
			return ErrProductDeletePendingOrder
		}
		if blockIncomplete {
			return ErrProductDeleteIncompleteOrder
		}
	}

	existing, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	imageURLs := productColorImageURLs(*existing)
	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	c.deleteOrphanedProductImages(imageURLs, nil)
	return nil
}

func validateProductInput(in ProductInput) error {
	if strings.TrimSpace(in.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if !in.Category.Valid() {
		return fmt.Errorf("invalid category")
	}
	if len(in.Colors) == 0 {
		return fmt.Errorf("at least one color is required")
	}
	if err := validateColorImages(normalizeStrings(in.Colors), in.ColorImages); err != nil {
		return err
	}
	if err := pricing.ValidateSizeVariants(in.Sizes); err != nil {
		return err
	}
	if err := validateTierDeliveryDays(in.Sizes); err != nil {
		return err
	}
	return nil
}

func validateTierDeliveryDays(sizes []model.SizeVariant) error {
	for _, sv := range sizes {
		if err := pricing.ValidateTierDeliveryDays(sv.Code, sv.Tiers); err != nil {
			return err
		}
	}
	return nil
}

func normalizeStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func normalizeColorImages(items []model.ColorImage) []model.ColorImage {
	out := make([]model.ColorImage, 0, len(items))
	for _, ci := range items {
		color := strings.TrimSpace(ci.Color)
		url := strings.TrimSpace(ci.ImageURL)
		if color == "" || url == "" {
			continue
		}
		out = append(out, model.ColorImage{Color: color, ImageURL: url})
	}
	if out == nil {
		return []model.ColorImage{}
	}
	return out
}

func validateColorImages(colors []string, images []model.ColorImage) error {
	byColor := make(map[string]string, len(images))
	for _, ci := range normalizeColorImages(images) {
		if _, dup := byColor[ci.Color]; dup {
			return fmt.Errorf("duplicate image for color %q", ci.Color)
		}
		byColor[ci.Color] = ci.ImageURL
	}
	for _, c := range colors {
		if byColor[c] == "" {
			return fmt.Errorf("image required for color %q", c)
		}
	}
	for c := range byColor {
		found := slices.Contains(colors, c)
		if !found {
			return fmt.Errorf("color image for unknown color %q", c)
		}
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}

func productColorImageURLs(p model.Product) []string {
	out := make([]string, 0, len(p.ColorImages))
	for _, ci := range p.ColorImages {
		u := strings.TrimSpace(ci.ImageURL)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func productImageURLsToDelete(previous, current []string) []string {
	keep := make(map[string]struct{}, len(current))
	for _, u := range current {
		keep[u] = struct{}{}
	}
	var delete []string
	for _, u := range previous {
		if _, ok := keep[u]; !ok {
			delete = append(delete, u)
		}
	}
	return delete
}

func (c *ProductController) deleteOrphanedProductImages(previous, current []string) {
	if c.images == nil {
		return
	}
	for _, u := range productImageURLsToDelete(previous, current) {
		if err := c.images.DeleteByPublicURL(u); err != nil {
			c.log.Warn("failed to delete orphaned product image", "url", u, "err", err)
			continue
		}
		c.log.Info("deleted orphaned product image", "url", u)
	}
}

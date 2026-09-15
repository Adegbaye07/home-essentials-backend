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
	Title           string
	Description     string
	Category        model.Category
	Variants        []string
	VariantImages   []model.VariantImage
	SizePricings    []model.SizePricing
	CleaningPricing *model.CleaningPricing
	Active          bool
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
	normalized, err := normalizeAndValidateProductInput(in)
	if err != nil {
		return nil, err
	}

	p := &model.Product{
		Title:           normalized.Title,
		Description:     normalized.Description,
		Category:        normalized.Category,
		Variants:        normalized.Variants,
		VariantImages:   normalized.VariantImages,
		SizePricings:    normalized.SizePricings,
		CleaningPricing: normalized.CleaningPricing,
		Active:          normalized.Active,
	}

	if err := c.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (c *ProductController) Update(ctx context.Context, id primitive.ObjectID, in ProductInput) (*model.Product, error) {
	normalized, err := normalizeAndValidateProductInput(in)
	if err != nil {
		return nil, err
	}

	existing, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	previousURLs := productVariantImageURLs(*existing)

	existing.Title = normalized.Title
	existing.Description = normalized.Description
	existing.Category = normalized.Category
	existing.Variants = normalized.Variants
	existing.VariantImages = normalized.VariantImages
	existing.SizePricings = normalized.SizePricings
	existing.CleaningPricing = normalized.CleaningPricing
	existing.Active = normalized.Active

	if err := c.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	c.deleteOrphanedProductImages(previousURLs, productVariantImageURLs(*existing))

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

	imageURLs := productVariantImageURLs(*existing)
	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	c.deleteOrphanedProductImages(imageURLs, nil)
	return nil
}

type normalizedProductInput struct {
	Title           string
	Description     string
	Category        model.Category
	Variants        []string
	VariantImages   []model.VariantImage
	SizePricings    []model.SizePricing
	CleaningPricing *model.CleaningPricing
	Active          bool
}

func normalizeAndValidateProductInput(in ProductInput) (normalizedProductInput, error) {
	out := normalizedProductInput{
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		Category:    in.Category,
		Active:      in.Active,
	}
	if out.Title == "" {
		return out, fmt.Errorf("title is required")
	}
	if !out.Category.Valid() {
		return out, fmt.Errorf("invalid category")
	}

	out.Variants = normalizeStrings(in.Variants)
	if len(out.Variants) == 0 {
		return out, fmt.Errorf("at least one variant is required")
	}
	out.VariantImages = normalizeVariantImages(in.VariantImages)
	if err := validateVariantImages(out.Variants, out.VariantImages); err != nil {
		return out, err
	}

	out.SizePricings = normalizeSizePricings(in.SizePricings)
	if in.CleaningPricing != nil {
		cp := *in.CleaningPricing
		out.CleaningPricing = &cp
	}

	if err := pricing.ValidateProductPricing(out.Category, out.SizePricings, out.CleaningPricing); err != nil {
		return out, err
	}

	if out.Category.IsCleaning() {
		out.SizePricings = nil
	} else {
		out.CleaningPricing = nil
	}

	return out, nil
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

func normalizeVariantImages(items []model.VariantImage) []model.VariantImage {
	out := make([]model.VariantImage, 0, len(items))
	for _, vi := range items {
		variant := strings.TrimSpace(vi.Variant)
		url := strings.TrimSpace(vi.ImageURL)
		if variant == "" || url == "" {
			continue
		}
		out = append(out, model.VariantImage{Variant: variant, ImageURL: url})
	}
	if out == nil {
		return []model.VariantImage{}
	}
	return out
}

func normalizeSizePricings(items []model.SizePricing) []model.SizePricing {
	out := make([]model.SizePricing, 0, len(items))
	for _, sp := range items {
		size := strings.TrimSpace(sp.Size)
		if size == "" {
			continue
		}
		out = append(out, model.SizePricing{
			Size:            size,
			PiecePriceKobo:  sp.PiecePriceKobo,
			BundlePriceKobo: sp.BundlePriceKobo,
			PiecesPerBundle: sp.PiecesPerBundle,
		})
	}
	if out == nil {
		return []model.SizePricing{}
	}
	return out
}

func validateVariantImages(variants []string, images []model.VariantImage) error {
	byVariant := make(map[string]string, len(images))
	for _, vi := range images {
		key := strings.ToLower(vi.Variant)
		if _, dup := byVariant[key]; dup {
			return fmt.Errorf("duplicate image for variant %q", vi.Variant)
		}
		byVariant[key] = vi.ImageURL
	}
	for _, v := range variants {
		if byVariant[strings.ToLower(v)] == "" {
			return fmt.Errorf("image required for variant %q", v)
		}
	}
	variantKeys := make([]string, 0, len(variants))
	for _, v := range variants {
		variantKeys = append(variantKeys, strings.ToLower(v))
	}
	for _, vi := range images {
		if !slices.Contains(variantKeys, strings.ToLower(vi.Variant)) {
			return fmt.Errorf("variant image for unknown variant %q", vi.Variant)
		}
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}

func productVariantImageURLs(p model.Product) []string {
	out := make([]string, 0, len(p.VariantImages))
	for _, vi := range p.VariantImages {
		u := strings.TrimSpace(vi.ImageURL)
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

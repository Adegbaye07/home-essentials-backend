package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pagination"
	"homeessentials/backend/internal/pricing"
	"homeessentials/backend/internal/repository"
)

type ProductHandler struct {
	ctrl *controller.ProductController
}

func NewProductHandler(ctrl *controller.ProductController) *ProductHandler {
	return &ProductHandler{ctrl: ctrl}
}

type variantImageDTO struct {
	Variant  string `json:"variant"`
	ImageURL string `json:"imageUrl"`
}

type sizePricingDTO struct {
	Size            string `json:"size"`
	PiecePriceKobo  int64  `json:"piecePriceKobo"`
	BundlePriceKobo int64  `json:"bundlePriceKobo"`
	PiecesPerBundle int    `json:"piecesPerBundle"`
}

type cleaningPricingDTO struct {
	PiecePriceKobo int64 `json:"piecePriceKobo"`
	DozenPriceKobo int64 `json:"dozenPriceKobo"`
}

type productRequest struct {
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	Category        string              `json:"category"`
	Variants        []string            `json:"variants"`
	VariantImages   []variantImageDTO   `json:"variantImages"`
	VideoURL        string              `json:"videoUrl"`
	SizePricings    []sizePricingDTO    `json:"sizePricings"`
	CleaningPricing *cleaningPricingDTO `json:"cleaningPricing"`
	Active          bool                `json:"active"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	in, err := requestToInput(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.ctrl.Create(c.Request.Context(), in)
	if err != nil {
		writeProductError(c, err)
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	in, err := requestToInput(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.ctrl.Update(c.Request.Context(), id, in)
	if err != nil {
		writeProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := h.ctrl.Delete(c.Request.Context(), id); err != nil {
		writeProductError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	p, err := h.ctrl.Get(c.Request.Context(), id)
	if err != nil {
		writeProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) List(c *gin.Context) {
	f, err := parseProductListFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.ctrl.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *ProductHandler) ListPublic(c *gin.Context) {
	f, err := parseProductListFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := true
	f.Active = &active

	out, err := h.ctrl.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, out)
}

func parseProductListFilter(c *gin.Context) (repository.ListFilter, error) {
	var f repository.ListFilter

	pg, err := pagination.ParseQuery(c.Query("page"), c.Query("page_size"))
	if err != nil {
		return f, err
	}
	f.Page = pg.Page
	f.PageSize = pg.PageSize

	if cat := c.Query("category"); cat != "" {
		parsed, err := model.ParseCategory(cat)
		if err != nil {
			return f, err
		}
		f.Category = &parsed
	}

	if activeStr := c.Query("active"); activeStr != "" {
		switch activeStr {
		case "true":
			t := true
			f.Active = &t
		case "false":
			fl := false
			f.Active = &fl
		default:
			return f, errors.New("active must be true or false")
		}
	}

	return f, nil
}

func (h *ProductHandler) GetPublic(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	p, err := h.ctrl.Get(c.Request.Context(), id)
	if err != nil {
		writeProductError(c, err)
		return
	}
	if !p.Active {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func requestToInput(req productRequest) (controller.ProductInput, error) {
	cat, err := model.ParseCategory(req.Category)
	if err != nil {
		return controller.ProductInput{}, err
	}

	variantImages := make([]model.VariantImage, 0, len(req.VariantImages))
	for _, vi := range req.VariantImages {
		variantImages = append(variantImages, model.VariantImage{
			Variant:  vi.Variant,
			ImageURL: vi.ImageURL,
		})
	}

	sizePricings := make([]model.SizePricing, 0, len(req.SizePricings))
	for _, sp := range req.SizePricings {
		sizePricings = append(sizePricings, model.SizePricing{
			Size:            sp.Size,
			PiecePriceKobo:  sp.PiecePriceKobo,
			BundlePriceKobo: sp.BundlePriceKobo,
			PiecesPerBundle: sp.PiecesPerBundle,
		})
	}

	var cleaning *model.CleaningPricing
	if req.CleaningPricing != nil {
		cleaning = &model.CleaningPricing{
			PiecePriceKobo: req.CleaningPricing.PiecePriceKobo,
			DozenPriceKobo: req.CleaningPricing.DozenPriceKobo,
		}
	}

	return controller.ProductInput{
		Title:           req.Title,
		Description:     req.Description,
		Category:        cat,
		Variants:        req.Variants,
		VariantImages:   variantImages,
		VideoURL:        req.VideoURL,
		SizePricings:    sizePricings,
		CleaningPricing: cleaning,
		Active:          req.Active,
	}, nil
}

func parseObjectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}

func writeProductError(c *gin.Context, err error) {
	if controller.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	if errors.Is(err, controller.ErrProductDeletePendingOrder) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "product_delete_pending_order"})
		return
	}
	if errors.Is(err, controller.ErrProductDeleteIncompleteOrder) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "product_delete_incomplete_order"})
		return
	}

	if errors.Is(err, pricing.ErrInvalidPricing) || errors.Is(err, pricing.ErrNoPriceForUnit) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if msg := err.Error(); msg == "title is required" ||
		msg == "invalid category" ||
		msg == "at least one variant is required" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if strings.Contains(err.Error(), "check orders for product delete") {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "product request failed"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

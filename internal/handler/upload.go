package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"homeessentials/backend/internal/controller"
)

const maxUploadBytes = 5 << 20

type UploadHandler struct {
	ctrl *controller.UploadController
	log  *slog.Logger
}

func NewUploadHandler(ctrl *controller.UploadController, log *slog.Logger) *UploadHandler {
	return &UploadHandler{ctrl: ctrl, log: log}
}

func (h *UploadHandler) UploadProductImage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxUploadBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer file.Close()

	url, err := h.ctrl.UploadProductImage(header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		switch {
		case errors.Is(err, controller.ErrUploadNotConfigured):
			h.log.Warn("upload not configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upload service not configured"})
		case errors.Is(err, controller.ErrInvalidImageType):
			h.log.Warn("upload invalid image type", "contentType", header.Header.Get("Content-Type"), "filename", header.Filename)
			c.JSON(http.StatusBadRequest, gin.H{"error": "allowed types: jpeg, png, webp, gif"})
		default:
			h.log.Error("upload failed",
				"err", err,
				"filename", header.Filename,
				"contentType", header.Header.Get("Content-Type"),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

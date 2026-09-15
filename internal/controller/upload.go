package controller

import (
	"fmt"
	"io"
	"strings"
)

var (
	ErrUploadNotConfigured = fmt.Errorf("image upload is not configured")
	ErrInvalidImageType    = fmt.Errorf("invalid image type")
)

type ImageUploader interface {
	UploadProductImage(filename string, r io.Reader, contentType string) (string, error)
}

type UploadController struct {
	uploader ImageUploader
}

func NewUploadController(uploader ImageUploader) *UploadController {
	return &UploadController{uploader: uploader}
}

func (c *UploadController) UploadProductImage(filename, contentType string, r io.Reader) (string, error) {
	if c.uploader == nil {
		return "", ErrUploadNotConfigured
	}

	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if !isAllowedImageType(contentType) {
		return "", ErrInvalidImageType
	}

	url, err := c.uploader.UploadProductImage(filename, r, contentType)
	if err != nil {
		return "", fmt.Errorf("upload product image: %w", err)
	}
	return url, nil
}

func isAllowedImageType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func imageContentTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	default:
		return ""
	}
}

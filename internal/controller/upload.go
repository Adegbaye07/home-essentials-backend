package controller

import (
	"fmt"
	"io"
	"strings"
)

var (
	ErrUploadNotConfigured = fmt.Errorf("image upload is not configured")
	ErrInvalidImageType    = fmt.Errorf("invalid image type")
	ErrInvalidVideoType    = fmt.Errorf("invalid video type")
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

	contentType = resolveImageContentType(filename, contentType)
	if !isAllowedImageType(contentType) {
		return "", ErrInvalidImageType
	}

	url, err := c.uploader.UploadProductImage(filename, r, contentType)
	if err != nil {
		return "", fmt.Errorf("upload product image: %w", err)
	}
	return url, nil
}

func (c *UploadController) UploadProductVideo(filename, contentType string, r io.Reader) (string, error) {
	if c.uploader == nil {
		return "", ErrUploadNotConfigured
	}

	contentType = resolveVideoContentType(filename, contentType)
	if !isAllowedVideoType(contentType) {
		return "", ErrInvalidVideoType
	}

	// Ensure object path gets a video extension when the client omits one.
	if !hasVideoExt(filename) {
		filename = filename + extForVideoContentType(contentType)
	}

	url, err := c.uploader.UploadProductImage(filename, r, contentType)
	if err != nil {
		return "", fmt.Errorf("upload product video: %w", err)
	}
	return url, nil
}

func resolveImageContentType(filename, contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if isAllowedImageType(contentType) {
		return contentType
	}
	if fromName := imageContentTypeFromFilename(filename); fromName != "" {
		return fromName
	}
	return contentType
}

func resolveVideoContentType(filename, contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	// Some clients send codecs in the type, e.g. video/mp4; codecs="avc1..."
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	if isAllowedVideoType(contentType) {
		return contentType
	}
	if fromName := videoContentTypeFromFilename(filename); fromName != "" {
		return fromName
	}
	return contentType
}

func isAllowedImageType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func isAllowedVideoType(ct string) bool {
	switch ct {
	case "video/mp4", "video/webm", "video/quicktime":
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

func videoContentTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".m4v"):
		return "video/mp4"
	case strings.HasSuffix(lower, ".webm"):
		return "video/webm"
	case strings.HasSuffix(lower, ".mov"):
		return "video/quicktime"
	default:
		return ""
	}
}

func hasVideoExt(filename string) bool {
	return videoContentTypeFromFilename(filename) != ""
}

func extForVideoContentType(ct string) string {
	switch ct {
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	default:
		return ".mp4"
	}
}

package storage

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	storagego "github.com/supabase-community/storage-go"
)

type Config struct {
	SupabaseURL    string
	ServiceRoleKey string
	Bucket         string
}

type Client struct {
	inner  *storagego.Client
	bucket string
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.SupabaseURL == "" || cfg.ServiceRoleKey == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("supabase storage is not configured")
	}

	baseURL := strings.TrimSuffix(cfg.SupabaseURL, "/") + "/storage/v1"
	return &Client{
		inner:  storagego.NewClient(baseURL, cfg.ServiceRoleKey, nil),
		bucket: cfg.Bucket,
	}, nil
}

func (c *Client) UploadProductImage(filename string, r io.Reader, contentType string) (string, error) {
	return c.uploadImage("products", filename, r, contentType)
}

func (c *Client) uploadImage(folder, filename string, r io.Reader, contentType string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}

	objectPath := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)

	upsert := false
	_, err := c.inner.UploadFile(c.bucket, objectPath, r, storagego.FileOptions{
		ContentType: &contentType,
		Upsert:      &upsert,
	})
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	public := c.inner.GetPublicUrl(c.bucket, objectPath)
	if public.SignedURL == "" {
		return "", fmt.Errorf("empty public url")
	}

	return public.SignedURL, nil
}

// DeleteByPublicURL removes a product image object previously uploaded to this bucket.
func (c *Client) DeleteByPublicURL(publicURL string) error {
	objectPath, err := objectPathFromPublicURL(publicURL, c.bucket)
	if err != nil {
		return err
	}
	_, err = c.inner.RemoveFile(c.bucket, []string{objectPath})
	if err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

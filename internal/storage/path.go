package storage

import (
	"fmt"
	"net/url"
	"strings"
)

const productObjectPrefix = "products/"

// objectPathFromPublicURL extracts the storage object path (e.g. products/uuid.jpg)
// from a Supabase public object URL for the configured bucket.
func objectPathFromPublicURL(publicURL, bucket string) (string, error) {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return "", fmt.Errorf("empty url")
	}

	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	// /storage/v1/object/public/{bucket}/{path...}
	marker := "/object/public/" + bucket + "/"
	idx := strings.Index(u.Path, marker)
	if idx < 0 {
		return "", fmt.Errorf("url is not a public object in bucket %q", bucket)
	}

	objectPath := strings.TrimPrefix(u.Path[idx:], marker)
	objectPath = strings.TrimPrefix(objectPath, "/")
	if objectPath == "" {
		return "", fmt.Errorf("empty object path")
	}

	if !strings.HasPrefix(objectPath, productObjectPrefix) {
		return "", fmt.Errorf("object path %q is outside %q", objectPath, productObjectPrefix)
	}

	return objectPath, nil
}

package controller

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type stubUploader struct {
	lastFilename    string
	lastContentType string
	url             string
	err             error
}

func (s *stubUploader) UploadProductImage(filename string, r io.Reader, contentType string) (string, error) {
	s.lastFilename = filename
	s.lastContentType = contentType
	_, _ = io.Copy(io.Discard, r)
	if s.err != nil {
		return "", s.err
	}
	return s.url, nil
}

func TestUploadProductVideo_ok(t *testing.T) {
	up := &stubUploader{url: "https://cdn.example/v.mp4"}
	ctrl := NewUploadController(up)
	got, err := ctrl.UploadProductVideo("clip.mp4", "video/mp4", strings.NewReader("fake"))
	if err != nil {
		t.Fatal(err)
	}
	if got != up.url {
		t.Fatalf("url: %q", got)
	}
	if up.lastContentType != "video/mp4" {
		t.Fatalf("contentType: %q", up.lastContentType)
	}
}

func TestUploadProductVideo_fromFilename(t *testing.T) {
	up := &stubUploader{url: "https://cdn.example/v.webm"}
	ctrl := NewUploadController(up)
	_, err := ctrl.UploadProductVideo("demo.webm", "application/octet-stream", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if up.lastContentType != "video/webm" {
		t.Fatalf("contentType: %q", up.lastContentType)
	}
}

func TestUploadProductVideo_rejectsBadType(t *testing.T) {
	ctrl := NewUploadController(&stubUploader{url: "x"})
	_, err := ctrl.UploadProductVideo("a.txt", "text/plain", strings.NewReader("x"))
	if !errors.Is(err, ErrInvalidVideoType) {
		t.Fatalf("got %v", err)
	}
}

func TestUploadProductVideo_notConfigured(t *testing.T) {
	ctrl := NewUploadController(nil)
	_, err := ctrl.UploadProductVideo("a.mp4", "video/mp4", strings.NewReader("x"))
	if !errors.Is(err, ErrUploadNotConfigured) {
		t.Fatalf("got %v", err)
	}
}

func TestUploadProductVideo_appendsExt(t *testing.T) {
	up := &stubUploader{url: "https://cdn.example/v.mp4"}
	ctrl := NewUploadController(up)
	_, err := ctrl.UploadProductVideo("clip", "video/mp4", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(up.lastFilename, ".mp4") {
		t.Fatalf("filename: %q", up.lastFilename)
	}
}

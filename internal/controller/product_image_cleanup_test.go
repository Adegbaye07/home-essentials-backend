package controller

import (
	"testing"

	"homeessentials/backend/internal/model"
)

func TestProductImageURLsToDelete(t *testing.T) {
	prev := []string{"https://x/a.jpg", "https://x/b.jpg", "https://x/c.jpg"}
	curr := []string{"https://x/b.jpg", "https://x/d.jpg"}
	got := productImageURLsToDelete(prev, curr)
	want := []string{"https://x/a.jpg", "https://x/c.jpg"}
	if len(got) != len(want) {
		t.Fatalf("len %d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestProductMediaURLs_includesVideo(t *testing.T) {
	p := model.Product{
		VariantImages: []model.VariantImage{
			{Variant: "beige", ImageURL: "https://x/a.jpg"},
		},
		VideoURL: "https://x/v.mp4",
	}
	got := productMediaURLs(p)
	if len(got) != 2 || got[0] != "https://x/a.jpg" || got[1] != "https://x/v.mp4" {
		t.Fatalf("got %v", got)
	}
}

func TestNormalizeProductInput_trimsVideoURL(t *testing.T) {
	in := validRugProductInput()
	in.VideoURL = "  https://cdn.example/a.mp4  "
	got, err := normalizeAndValidateProductInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.VideoURL != "https://cdn.example/a.mp4" {
		t.Fatalf("videoUrl: %q", got.VideoURL)
	}
}

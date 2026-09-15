package controller

import "testing"

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

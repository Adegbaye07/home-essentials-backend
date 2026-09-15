package model

import "testing"

func TestParseCategory_newSlug(t *testing.T) {
	got, err := ParseCategory("clutch")
	if err != nil {
		t.Fatal(err)
	}
	if got != CategoryClutch {
		t.Fatalf("got %q, want %q", got, CategoryClutch)
	}
}

func TestParseCategory_legacyMens(t *testing.T) {
	got, err := ParseCategory("mens")
	if err != nil {
		t.Fatal(err)
	}
	if got != CategoryMens {
		t.Fatalf("got %q, want %q", got, CategoryMens)
	}
}

func TestParseCategory_handBag(t *testing.T) {
	got, err := ParseCategory("hand_bag")
	if err != nil {
		t.Fatal(err)
	}
	if got != CategoryHandBag {
		t.Fatalf("got %q, want %q", got, CategoryHandBag)
	}
}

func TestParseCategory_flapBag(t *testing.T) {
	got, err := ParseCategory("flap_bag")
	if err != nil {
		t.Fatal(err)
	}
	if got != CategoryFlapBag {
		t.Fatalf("got %q, want %q", got, CategoryFlapBag)
	}
}

func TestParseCategory_existingTote(t *testing.T) {
	got, err := ParseCategory("tote")
	if err != nil {
		t.Fatal(err)
	}
	if got != CategoryTote {
		t.Fatalf("got %q, want %q", got, CategoryTote)
	}
}

func TestParseCategory_invalid(t *testing.T) {
	_, err := ParseCategory("not-a-cat")
	if err == nil {
		t.Fatal("expected error")
	}
}

package model

import "testing"

func TestParseCategory_homeEssentials(t *testing.T) {
	cases := []struct {
		in   string
		want Category
	}{
		{"foot_mats", CategoryFootMats},
		{"door_mats", CategoryDoorMats},
		{"center_mats", CategoryCenterMats},
		{"rugs", CategoryRugs},
		{"cleaning_essentials", CategoryCleaningEssentials},
	}
	for _, tc := range cases {
		got, err := ParseCategory(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseCategory_invalid(t *testing.T) {
	_, err := ParseCategory("tote")
	if err == nil {
		t.Fatal("expected error for legacy bag category")
	}
}

func TestCategory_IsCleaning(t *testing.T) {
	if !CategoryCleaningEssentials.IsCleaning() {
		t.Fatal("expected cleaning")
	}
	if CategoryRugs.IsCleaning() {
		t.Fatal("rugs should not be cleaning")
	}
}

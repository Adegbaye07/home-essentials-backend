package pagination_test

import (
	"testing"

	"homeessentials/backend/internal/pagination"
)

func TestParseQuery_defaults(t *testing.T) {
	p, err := pagination.ParseQuery("", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Page != 1 || p.PageSize != 20 {
		t.Fatalf("got page=%d pageSize=%d", p.Page, p.PageSize)
	}
}

func TestParseQuery_custom(t *testing.T) {
	p, err := pagination.ParseQuery("3", "10")
	if err != nil {
		t.Fatal(err)
	}
	if p.Page != 3 || p.PageSize != 10 {
		t.Fatalf("got page=%d pageSize=%d", p.Page, p.PageSize)
	}
	if p.Skip() != 20 {
		t.Fatalf("skip = %d", p.Skip())
	}
}

func TestParseQuery_invalid(t *testing.T) {
	if _, err := pagination.ParseQuery("0", ""); err == nil {
		t.Fatal("want error for page 0")
	}
	if _, err := pagination.ParseQuery("", "101"); err == nil {
		t.Fatal("want error for page_size > max")
	}
}

func TestBuildMetadata_firstPage(t *testing.T) {
	m := pagination.BuildMetadata(45, 1, 20, 20)
	if m.TotalItems != 45 || m.CurrentItems != 20 || m.CurrentPage != 1 || m.LastPage != 3 {
		t.Fatalf("%+v", m)
	}
	if !m.HasNextPage || m.HasPreviousPage || m.NextPage == nil || *m.NextPage != 2 {
		t.Fatalf("next/prev wrong: %+v", m)
	}
	if m.PreviousPage != nil {
		t.Fatalf("previous should be nil")
	}
}

func TestBuildMetadata_lastPagePartial(t *testing.T) {
	m := pagination.BuildMetadata(45, 3, 20, 5)
	if m.CurrentItems != 5 || !m.HasPreviousPage || m.HasNextPage {
		t.Fatalf("%+v", m)
	}
	if m.PreviousPage == nil || *m.PreviousPage != 2 {
		t.Fatalf("previous page")
	}
}

func TestBuildMetadata_emptyTotal(t *testing.T) {
	m := pagination.BuildMetadata(0, 1, 20, 0)
	if m.LastPage != 1 || m.TotalItems != 0 || m.HasNextPage {
		t.Fatalf("%+v", m)
	}
}

func TestBuildMetadata_pageBeyondLast(t *testing.T) {
	m := pagination.BuildMetadata(10, 5, 20, 0)
	if m.CurrentPage != 5 || m.CurrentItems != 0 || m.LastPage != 1 {
		t.Fatalf("%+v", m)
	}
	if m.HasNextPage {
		t.Fatal("should not have next page")
	}
}

func TestNewPaginated_nilItems(t *testing.T) {
	out := pagination.NewPaginated([]string(nil), 0, 1, 20)
	if out.Items == nil || len(out.Items) != 0 {
		t.Fatalf("want empty non-nil slice")
	}
}

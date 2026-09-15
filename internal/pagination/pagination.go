package pagination

import (
	"fmt"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Params holds validated page query values.
type Params struct {
	Page     int
	PageSize int
}

// Metadata describes the current page in a paginated list response.
type Metadata struct {
	TotalItems      int64 `json:"total_items"`
	CurrentItems    int   `json:"current_items"`
	CurrentPage     int   `json:"current_page"`
	LastPage        int   `json:"last_page"`
	NextPage        *int  `json:"next_page"`
	PreviousPage    *int  `json:"previous_page"`
	HasNextPage     bool  `json:"has_next_page"`
	HasPreviousPage bool  `json:"has_previous_page"`
}

// Paginated is the standard list API response shape.
type Paginated[T any] struct {
	Items    []T      `json:"items"`
	Metadata Metadata `json:"metadata"`
}

// ParseQuery reads page and page_size from query strings (empty = defaults).
func ParseQuery(pageStr, pageSizeStr string) (Params, error) {
	page := DefaultPage
	pageSize := DefaultPageSize

	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return Params{}, fmt.Errorf("page must be a positive integer")
		}
		page = p
	}

	if pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 || ps > MaxPageSize {
			return Params{}, fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)
		}
		pageSize = ps
	}

	return Params{Page: page, PageSize: pageSize}, nil
}

// Skip returns MongoDB skip offset for the page.
func (p Params) Skip() int64 {
	return int64((p.Page - 1) * p.PageSize)
}

// Limit returns MongoDB limit for the page.
func (p Params) Limit() int64 {
	return int64(p.PageSize)
}

// BuildMetadata computes list metadata. currentCount is len(items) on this page.
func BuildMetadata(total int64, page, pageSize, currentCount int) Metadata {
	lastPage := LastPage(total, pageSize)

	hasNext := page < lastPage
	hasPrev := page > 1

	var nextPage, prevPage *int
	if hasNext {
		n := page + 1
		nextPage = &n
	}
	if hasPrev {
		p := page - 1
		prevPage = &p
	}

	return Metadata{
		TotalItems:      total,
		CurrentItems:    currentCount,
		CurrentPage:     page,
		LastPage:        lastPage,
		NextPage:        nextPage,
		PreviousPage:    prevPage,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}
}

// LastPage returns the last page number for total items (minimum 1).
func LastPage(total int64, pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if total <= 0 {
		return 1
	}
	lp := int((total + int64(pageSize) - 1) / int64(pageSize))
	if lp < 1 {
		return 1
	}
	return lp
}

// NewPaginated builds a response with non-nil items slice.
func NewPaginated[T any](items []T, total int64, page, pageSize int) Paginated[T] {
	if items == nil {
		items = []T{}
	}
	return Paginated[T]{
		Items:    items,
		Metadata: BuildMetadata(total, page, pageSize, len(items)),
	}
}

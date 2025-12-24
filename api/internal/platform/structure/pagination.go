package structure

import "math"

// PaginationParams is a generic pagination request with query filter.
type PaginationParams[Q any] struct {
	Page     int32
	PageSize int32
	Query    Q
}

// SetDefaults applies default values for pagination.
func (p *PaginationParams[Q]) SetDefaults() {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}
}

// Skip returns the number of items to skip for MongoDB queries.
func (p *PaginationParams[Q]) Skip() int64 {
	return int64((p.Page - 1) * p.PageSize)
}

// PaginationMeta contains pagination response metadata.
type PaginationMeta struct {
	CurrentPage int32
	PageSize    int32
	TotalPages  int32
	TotalCount  int64
}

// NewPaginationMeta creates pagination metadata from page, pageSize and total count.
func NewPaginationMeta(page, pageSize int32, totalCount int64) PaginationMeta {
	totalPages := int32(math.Ceil(float64(totalCount) / float64(pageSize)))
	return PaginationMeta{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
}

// PaginatedResult is a generic wrapper for paginated results.
type PaginatedResult[T any] struct {
	Items []T
	PaginationMeta
}

// NewPaginatedResult creates a new paginated result.
func NewPaginatedResult[T any](items []T, page, pageSize int32, totalCount int64) *PaginatedResult[T] {
	return &PaginatedResult[T]{
		Items:          items,
		PaginationMeta: NewPaginationMeta(page, pageSize, totalCount),
	}
}

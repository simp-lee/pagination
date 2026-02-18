package pagination

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidPageNumber = errors.New("page number must be greater than 0")
	ErrCallbackNotFound  = errors.New("callback function not found")
	ErrInvalidConfig     = errors.New("invalid paginator configuration")
)

// Paginator handles type-safe pagination logic.
type Paginator[T any] struct {
	// itemTotalCallback returns the total number of items
	itemTotalCallback func(ctx context.Context) (int64, error)
	// sliceCallback returns a slice of items for the current page
	sliceCallback func(ctx context.Context, offset, limit int) ([]T, error)
	// itemsPerPage defines how many items to display per page
	itemsPerPage int
	// pagesInRange defines how many page numbers to show in navigation
	pagesInRange int
	// configErr captures invalid option configuration errors
	configErr error
	// knownTotal optionally provides a precomputed total to skip itemTotalCallback
	knownTotal *int64
}

// Option configures Paginator.
type Option[T any] func(*Paginator[T])

// NewPaginator creates a new type-safe paginator instance with the given options.
func NewPaginator[T any](config ...Option[T]) *Paginator[T] {
	p := &Paginator[T]{
		itemsPerPage: 10,
		pagesInRange: 5,
	}

	for _, opt := range config {
		if opt == nil {
			continue
		}
		opt(p)
	}

	return p
}

func (p *Paginator[T]) setConfigError(err error) {
	p.configErr = errors.Join(p.configErr, err)
}

// WithItemsPerPage sets the number of items per page for Paginator.
func WithItemsPerPage[T any](n int) Option[T] {
	return func(p *Paginator[T]) {
		if n <= 0 {
			p.setConfigError(fmt.Errorf("%w: items per page must be greater than 0", ErrInvalidConfig))
			return
		}
		p.itemsPerPage = n
	}
}

// WithPagesInRange sets the number of page numbers to show in navigation for Paginator.
func WithPagesInRange[T any](n int) Option[T] {
	return func(p *Paginator[T]) {
		if n <= 0 {
			p.setConfigError(fmt.Errorf("%w: pages in range must be greater than 0", ErrInvalidConfig))
			return
		}
		p.pagesInRange = n
	}
}

// WithItemTotalCallback sets the callback function for getting total items count for Paginator.
func WithItemTotalCallback[T any](cb func(ctx context.Context) (int64, error)) Option[T] {
	return func(p *Paginator[T]) {
		p.itemTotalCallback = cb
	}
}

// WithSliceCallback sets the callback function for getting page items for Paginator.
func WithSliceCallback[T any](cb func(ctx context.Context, offset, limit int) ([]T, error)) Option[T] {
	return func(p *Paginator[T]) {
		p.sliceCallback = cb
	}
}

// WithKnownTotal sets a precomputed total items count and skips itemTotalCallback in Paginate.
func WithKnownTotal[T any](total int64) Option[T] {
	return func(p *Paginator[T]) {
		if total < 0 {
			p.setConfigError(fmt.Errorf("%w: total items must be greater than or equal to 0", ErrInvalidConfig))
			return
		}
		p.knownTotal = &total
	}
}

// Paginate performs pagination and returns a type-safe result.
func (p *Paginator[T]) Paginate(ctx context.Context, currentPage int) (*Pagination[T], error) {
	if p.configErr != nil {
		return nil, p.configErr
	}
	if p.sliceCallback == nil {
		return nil, fmt.Errorf("%w: sliceCallback is required", ErrCallbackNotFound)
	}
	if p.knownTotal == nil && p.itemTotalCallback == nil {
		return nil, fmt.Errorf("%w: either itemTotalCallback or knownTotal is required", ErrCallbackNotFound)
	}
	if currentPage <= 0 {
		return nil, ErrInvalidPageNumber
	}

	total := int64(0)
	var err error
	if p.knownTotal != nil {
		total = *p.knownTotal
	} else {
		total, err = p.itemTotalCallback(ctx)
		if err != nil {
			return nil, fmt.Errorf("itemTotalCallback failed: %w", err)
		}
	}
	if total < 0 {
		return nil, fmt.Errorf("itemTotalCallback failed: %w", ErrInvalidConfig)
	}

	perPage := int64(p.itemsPerPage)
	numberOfPages := int(total / perPage)
	if total%perPage != 0 {
		numberOfPages++
	}
	if numberOfPages == 0 {
		numberOfPages = 1
	}

	if currentPage > numberOfPages {
		currentPage = numberOfPages
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	offset := (currentPage - 1) * p.itemsPerPage
	items, err := p.sliceCallback(ctx, offset, p.itemsPerPage)
	if err != nil {
		return nil, fmt.Errorf("sliceCallback failed: %w", err)
	}
	if items == nil {
		items = make([]T, 0)
	}

	pages := calculatePageRange(currentPage, numberOfPages, p.pagesInRange)

	pagination := &Pagination[T]{
		Items:            items,
		Pages:            pages,
		TotalPages:       numberOfPages,
		CurrentPage:      currentPage,
		FirstPage:        1,
		LastPage:         numberOfPages,
		ItemsPerPage:     p.itemsPerPage,
		TotalItems:       total,
		FirstPageInRange: pages[0],
		LastPageInRange:  pages[len(pages)-1],
	}

	if currentPage > 1 {
		prev := currentPage - 1
		pagination.PreviousPage = &prev
	}
	if currentPage < numberOfPages {
		next := currentPage + 1
		pagination.NextPage = &next
	}

	return pagination, nil
}

// calculatePageRange calculates which page numbers to show in navigation.
func calculatePageRange(currentPage, totalPages, windowSize int) []int {
	if totalPages <= windowSize {
		return generateSequence(1, totalPages)
	}

	start := currentPage - (windowSize-1)/2
	end := start + windowSize - 1

	if start < 1 {
		start = 1
		end = windowSize
	}
	if end > totalPages {
		end = totalPages
		start = totalPages - windowSize + 1
	}

	return generateSequence(start, end)
}

// generateSequence generates a sequence of numbers from start to end inclusive
func generateSequence(start, end int) []int {
	if start > end {
		return []int{}
	}

	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}

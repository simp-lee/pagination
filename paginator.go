package pagination

import (
	"context"
	"errors"
	"math"
)

var (
	ErrInvalidPageNumber = errors.New("page number must be greater than 0")
	ErrCallbackNotFound  = errors.New("callback function not found")
)

// Paginator handles the pagination logic
type Paginator struct {
	// itemTotalCallback returns the total number of items
	itemTotalCallback func(ctx context.Context) (int64, error)
	// sliceCallback returns a slice of items for the current page
	sliceCallback func(ctx context.Context, offset, limit int) (interface{}, error)
	// itemsPerPage defines how many items to display per page
	itemsPerPage int
	// pagesInRange defines how many page numbers to show in navigation
	pagesInRange int
}

// NewPaginator creates a new Paginator instance with the given options
func NewPaginator(config ...Option) *Paginator {
	p := &Paginator{
		itemsPerPage: 10, // default 10 items per page
		pagesInRange: 5,  // default 5 page numbers in navigation
	}

	for _, opt := range config {
		opt(p)
	}

	return p
}

type Option func(*Paginator)

// WithItemsPerPage sets the number of items per page
func WithItemsPerPage(n int) Option {
	return func(p *Paginator) {
		if n <= 0 {
			panic("items per page must be greater than 0")
		}
		p.itemsPerPage = n
	}
}

// WithPagesInRange sets the number of page numbers to show in navigation
func WithPagesInRange(n int) Option {
	return func(p *Paginator) {
		if n <= 0 {
			panic("pages in range must be greater than 0")
		}
		p.pagesInRange = n
	}
}

// WithItemTotalCallback sets the callback function for getting total items count
func WithItemTotalCallback(cb func(ctx context.Context) (int64, error)) Option {
	return func(p *Paginator) {
		p.itemTotalCallback = cb
	}
}

// WithSliceCallback sets the callback function for getting page items
func WithSliceCallback(cb func(ctx context.Context, offset, limit int) (interface{}, error)) Option {
	return func(p *Paginator) {
		p.sliceCallback = cb
	}
}

// Paginate performs the pagination and returns the result
func (p *Paginator) Paginate(ctx context.Context, currentPage int) (*Pagination, error) {
	if p.itemTotalCallback == nil || p.sliceCallback == nil {
		return nil, ErrCallbackNotFound
	}
	if currentPage <= 0 {
		return nil, ErrInvalidPageNumber
	}

	// Get total items count
	total, err := p.itemTotalCallback(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	numberOfPages := int(math.Ceil(float64(total) / float64(p.itemsPerPage)))
	if numberOfPages == 0 {
		numberOfPages = 1
	}

	// Ensure current page doesn't exceed total pages
	if currentPage > numberOfPages {
		currentPage = numberOfPages
	}

	// Calculate offset and get page items
	offset := (currentPage - 1) * p.itemsPerPage
	items, err := p.sliceCallback(ctx, offset, p.itemsPerPage)
	if err != nil {
		return nil, err
	}

	// Calculate page range for navigation
	pages := p.calculatePageRange(currentPage, numberOfPages)

	// Build pagination result
	pagination := &Pagination{
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

	// Set previous/next page
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

// calculatePageRange calculates which page numbers to show in navigation
func (p *Paginator) calculatePageRange(currentPage, totalPages int) []int {
	if totalPages <= p.pagesInRange {
		return generateSequence(1, totalPages)
	}

	half := p.pagesInRange / 2
	start := currentPage - half
	end := currentPage + half

	// Handle edge cases
	if start < 1 {
		start = 1
		end = p.pagesInRange
	}
	if end > totalPages {
		end = totalPages
		start = totalPages - p.pagesInRange + 1
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

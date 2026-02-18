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
	ErrInvalidCursor     = errors.New("invalid cursor request")
	ErrInvalidKeyset     = errors.New("invalid keyset request")
)

// Direction indicates traversal direction for cursor/keyset requests.
type Direction string

const (
	DirectionForward  Direction = "forward"
	DirectionBackward Direction = "backward"
)

// CursorRequest defines a cursor-based pagination request.
type CursorRequest struct {
	AfterCursor  *string
	BeforeCursor *string
	Limit        int
	Direction    Direction
}

// CursorResult defines cursor callback output.
type CursorResult[T any] struct {
	Items          []T
	NextCursor     *string
	PreviousCursor *string
	HasMore        bool
}

// KeysetRequest defines a keyset-based pagination request.
type KeysetRequest struct {
	AfterKey  *string
	BeforeKey *string
	Limit     int
	Direction Direction
}

// KeysetResult defines keyset callback output.
type KeysetResult[T any] struct {
	Items       []T
	NextKey     *string
	PreviousKey *string
	HasMore     bool
}

// Paginator handles type-safe pagination logic.
type Paginator[T any] struct {
	// itemTotalCallback returns the total number of items
	itemTotalCallback func(ctx context.Context) (int64, error)
	// sliceCallback returns a slice of items for the current page
	sliceCallback func(ctx context.Context, offset, limit int) ([]T, error)
	// cursorSliceCallback returns a cursor-segment result
	cursorSliceCallback func(ctx context.Context, req CursorRequest) (*CursorResult[T], error)
	// keysetSliceCallback returns a keyset-segment result
	keysetSliceCallback func(ctx context.Context, req KeysetRequest) (*KeysetResult[T], error)
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
		if cb == nil {
			p.setConfigError(fmt.Errorf("%w: itemTotalCallback must not be nil", ErrInvalidConfig))
			return
		}
		p.itemTotalCallback = cb
	}
}

// WithSliceCallback sets the callback function for getting page items for Paginator.
func WithSliceCallback[T any](cb func(ctx context.Context, offset, limit int) ([]T, error)) Option[T] {
	return func(p *Paginator[T]) {
		if cb == nil {
			p.setConfigError(fmt.Errorf("%w: sliceCallback must not be nil", ErrInvalidConfig))
			return
		}
		p.sliceCallback = cb
	}
}

// WithCursorSliceCallback sets the callback for cursor-based pagination.
func WithCursorSliceCallback[T any](cb func(ctx context.Context, req CursorRequest) (*CursorResult[T], error)) Option[T] {
	return func(p *Paginator[T]) {
		if cb == nil {
			p.setConfigError(fmt.Errorf("%w: cursorSliceCallback must not be nil", ErrInvalidConfig))
			return
		}
		p.cursorSliceCallback = cb
	}
}

// WithKeysetSliceCallback sets the callback for keyset-based pagination.
func WithKeysetSliceCallback[T any](cb func(ctx context.Context, req KeysetRequest) (*KeysetResult[T], error)) Option[T] {
	return func(p *Paginator[T]) {
		if cb == nil {
			p.setConfigError(fmt.Errorf("%w: keysetSliceCallback must not be nil", ErrInvalidConfig))
			return
		}
		p.keysetSliceCallback = cb
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
	if ctx == nil {
		return nil, fmt.Errorf("%w: context must not be nil", ErrInvalidConfig)
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
	if err := ctx.Err(); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("total items must not be negative: %w", ErrInvalidConfig)
	}

	perPage := int64(p.itemsPerPage)
	numberOfPages64 := total / perPage
	if total%perPage != 0 {
		numberOfPages64++
	}
	if numberOfPages64 == 0 {
		numberOfPages64 = 1
	}

	maxInt := int64(^uint(0) >> 1)
	if numberOfPages64 > maxInt {
		return nil, fmt.Errorf("%w: total pages exceed int range", ErrInvalidConfig)
	}
	numberOfPages := int(numberOfPages64)

	if currentPage > numberOfPages {
		currentPage = numberOfPages
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	offset64 := int64(currentPage-1) * int64(p.itemsPerPage)
	if offset64 > maxInt {
		return nil, fmt.Errorf("%w: offset exceeds int range", ErrInvalidConfig)
	}
	offset := int(offset64)
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

// PaginateByCursor performs cursor-based pagination and returns a type-safe result.
func (p *Paginator[T]) PaginateByCursor(ctx context.Context, req CursorRequest) (*CursorPagination[T], error) {
	if p.configErr != nil {
		return nil, p.configErr
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context must not be nil", ErrInvalidConfig)
	}
	if p.cursorSliceCallback == nil {
		return nil, fmt.Errorf("%w: cursorSliceCallback is required", ErrCallbackNotFound)
	}

	normalized, err := p.normalizeCursorRequest(req)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result, err := p.cursorSliceCallback(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("cursorSliceCallback failed: %w", err)
	}

	items := make([]T, 0)
	if result != nil && result.Items != nil {
		items = result.Items
	}

	pagination := &CursorPagination[T]{
		Items:     items,
		Limit:     normalized.Limit,
		Direction: normalized.Direction,
	}

	if result != nil {
		pagination.NextCursor = result.NextCursor
		pagination.PreviousCursor = result.PreviousCursor
		pagination.HasMore = result.HasMore
	}

	return pagination, nil
}

// PaginateByKeyset performs keyset-based pagination and returns a type-safe result.
func (p *Paginator[T]) PaginateByKeyset(ctx context.Context, req KeysetRequest) (*KeysetPagination[T], error) {
	if p.configErr != nil {
		return nil, p.configErr
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context must not be nil", ErrInvalidConfig)
	}
	if p.keysetSliceCallback == nil {
		return nil, fmt.Errorf("%w: keysetSliceCallback is required", ErrCallbackNotFound)
	}

	normalized, err := p.normalizeKeysetRequest(req)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result, err := p.keysetSliceCallback(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("keysetSliceCallback failed: %w", err)
	}

	items := make([]T, 0)
	if result != nil && result.Items != nil {
		items = result.Items
	}

	pagination := &KeysetPagination[T]{
		Items:     items,
		Limit:     normalized.Limit,
		Direction: normalized.Direction,
	}

	if result != nil {
		pagination.NextKey = result.NextKey
		pagination.PreviousKey = result.PreviousKey
		pagination.HasMore = result.HasMore
	}

	return pagination, nil
}

func (p *Paginator[T]) normalizeCursorRequest(req CursorRequest) (CursorRequest, error) {
	if req.AfterCursor != nil && req.BeforeCursor != nil {
		return CursorRequest{}, fmt.Errorf("%w: after_cursor and before_cursor cannot be set together", ErrInvalidCursor)
	}
	if req.Limit < 0 {
		return CursorRequest{}, fmt.Errorf("%w: limit must not be negative", ErrInvalidCursor)
	}
	if req.Limit == 0 {
		req.Limit = p.itemsPerPage
	}
	if req.Direction == "" {
		req.Direction = DirectionForward
	}
	if req.Direction != DirectionForward && req.Direction != DirectionBackward {
		return CursorRequest{}, fmt.Errorf("%w: unsupported direction %q", ErrInvalidCursor, req.Direction)
	}
	return req, nil
}

func (p *Paginator[T]) normalizeKeysetRequest(req KeysetRequest) (KeysetRequest, error) {
	if req.AfterKey != nil && req.BeforeKey != nil {
		return KeysetRequest{}, fmt.Errorf("%w: after_key and before_key cannot be set together", ErrInvalidKeyset)
	}
	if req.Limit < 0 {
		return KeysetRequest{}, fmt.Errorf("%w: limit must not be negative", ErrInvalidKeyset)
	}
	if req.Limit == 0 {
		req.Limit = p.itemsPerPage
	}
	if req.Direction == "" {
		req.Direction = DirectionForward
	}
	if req.Direction != DirectionForward && req.Direction != DirectionBackward {
		return KeysetRequest{}, fmt.Errorf("%w: unsupported direction %q", ErrInvalidKeyset, req.Direction)
	}
	return req, nil
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

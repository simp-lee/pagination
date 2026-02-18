package pagination

// Pagination represents a type-safe pagination result structure.
type Pagination[T any] struct {
	// Items contains the slice of current page items
	Items []T `json:"items"`

	// Pages contains the array of page numbers to be displayed
	Pages []int `json:"pages"`

	// TotalPages is the total number of pages
	TotalPages int `json:"total_pages"`

	// CurrentPage is the current page number
	CurrentPage int `json:"current_page"`

	// FirstPage is always 1
	FirstPage int `json:"first_page"`

	// LastPage is equal to TotalPages
	LastPage int `json:"last_page"`

	// PreviousPage contains the previous page number, nil if current page is first page
	PreviousPage *int `json:"previous_page"`

	// NextPage contains the next page number, nil if current page is last page
	NextPage *int `json:"next_page"`

	// ItemsPerPage is the number of items per page
	ItemsPerPage int `json:"items_per_page"`

	// TotalItems is the total number of items across all pages
	TotalItems int64 `json:"total_items"`

	// FirstPageInRange is the first page number in the current page range
	FirstPageInRange int `json:"first_page_in_range"`

	// LastPageInRange is the last page number in the current page range
	LastPageInRange int `json:"last_page_in_range"`
}

// CursorPagination represents a cursor-based pagination result structure.
type CursorPagination[T any] struct {
	// Items contains the fetched items
	Items []T `json:"items"`

	// NextCursor is used to request the next segment
	NextCursor *string `json:"next_cursor"`

	// PreviousCursor is used to request the previous segment
	PreviousCursor *string `json:"previous_cursor"`

	// HasMore indicates whether more data is available in the requested direction
	HasMore bool `json:"has_more"`

	// Limit is the effective page size for this cursor request
	Limit int `json:"limit"`

	// Direction is the cursor traversal direction (forward/backward)
	Direction Direction `json:"direction"`
}

// KeysetPagination represents a keyset-based pagination result structure.
type KeysetPagination[T any] struct {
	// Items contains the fetched items
	Items []T `json:"items"`

	// NextKey is used to request the next segment
	NextKey *string `json:"next_key"`

	// PreviousKey is used to request the previous segment
	PreviousKey *string `json:"previous_key"`

	// HasMore indicates whether more data is available in the requested direction
	HasMore bool `json:"has_more"`

	// Limit is the effective page size for this keyset request
	Limit int `json:"limit"`

	// Direction is the keyset traversal direction (forward/backward)
	Direction Direction `json:"direction"`
}

// HasPreviousPage checks if there is a previous page available
func (p *Pagination[T]) HasPreviousPage() bool {
	return p.PreviousPage != nil
}

// HasNextPage checks if there is a next page available
func (p *Pagination[T]) HasNextPage() bool {
	return p.NextPage != nil
}

// IsFirstPage checks if the current page is the first page
func (p *Pagination[T]) IsFirstPage() bool {
	return p.CurrentPage == p.FirstPage
}

// IsLastPage checks if the current page is the last page
func (p *Pagination[T]) IsLastPage() bool {
	return p.CurrentPage == p.LastPage
}

// HasNext reports whether a NextCursor token is available for a subsequent
// forward request. This differs from the HasMore field, which is the
// authoritative signal from the data source indicating more data exists.
func (p *CursorPagination[T]) HasNext() bool {
	return p.NextCursor != nil
}

// HasPrevious reports whether a PreviousCursor token is available for a
// subsequent backward request. This differs from the HasMore field, which
// reflects the data source's own "more data" signal.
func (p *CursorPagination[T]) HasPrevious() bool {
	return p.PreviousCursor != nil
}

// HasNext reports whether a NextKey token is available for a subsequent
// forward request. This differs from the HasMore field, which is the
// authoritative signal from the data source indicating more data exists.
func (p *KeysetPagination[T]) HasNext() bool {
	return p.NextKey != nil
}

// HasPrevious reports whether a PreviousKey token is available for a
// subsequent backward request. This differs from the HasMore field, which
// reflects the data source's own "more data" signal.
func (p *KeysetPagination[T]) HasPrevious() bool {
	return p.PreviousKey != nil
}

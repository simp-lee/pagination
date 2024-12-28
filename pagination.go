package pagination

// Pagination represents the pagination result structure
type Pagination struct {
	// Items contains the slice of current page items
	Items interface{} `json:"items"`

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

// HasPreviousPage checks if there is a previous page available
func (p *Pagination) HasPreviousPage() bool {
	return p.PreviousPage != nil
}

// HasNextPage checks if there is a next page available
func (p *Pagination) HasNextPage() bool {
	return p.NextPage != nil
}

// IsFirstPage checks if the current page is the first page
func (p *Pagination) IsFirstPage() bool {
	return p.CurrentPage == p.FirstPage
}

// IsLastPage checks if the current page is the last page
func (p *Pagination) IsLastPage() bool {
	return p.CurrentPage == p.LastPage
}

// GetPageInfo returns a simplified map of pagination information
func (p *Pagination) GetPageInfo() map[string]interface{} {
	return map[string]interface{}{
		"current_page":   p.CurrentPage,
		"total_pages":    p.TotalPages,
		"total_items":    p.TotalItems,
		"has_next":       p.HasNextPage(),
		"has_previous":   p.HasPreviousPage(),
		"items_per_page": p.ItemsPerPage,
	}
}

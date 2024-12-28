# Pagination

A flexible and feature-rich pagination package for Go applications. Simple Pagination implements a paging interface on collections of things - from simple arrays to database lists to any collection you want to paginate through.

> Inspired by [AshleyDawson/SimplePagination](https://github.com/AshleyDawson/SimplePagination)

## Features

- Context support for database operations
- Customizable items per page
- Configurable page range navigation
- JSON-ready pagination results
- Helper methods for pagination state
- Callback-based data fetching

## Installation

```bash
go get github.com/simp-lee/pagination
```

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/simp-lee/pagination"
)

func main() {
    // Create a new paginator with custom options
    paginator := pagination.NewPaginator(
        pagination.WithItemsPerPage(10),
        pagination.WithPagesInRange(5),
        pagination.WithItemTotalCallback(func(ctx context.Context) (int64, error) {
            return 100, nil // Return total number of items
        }),
        pagination.WithSliceCallback(func(ctx context.Context, offset, limit int) (interface{}, error) {
            return []string{"item1", "item2", "item3", "item4", "item5", "item6", "item7", "item8", "item9", "item10"}, nil // Return slice of items for current page
        }),
    )
    
    // Get page 1
    result, err := paginator.Paginate(context.Background(), 1)
    if err != nil {
        panic(err)
    }

    // Get the actual items
    items := result.Items.([]string)
    fmt.Println("Items:", items)

    // Print pagination info
    info := result.GetPageInfo()
    bytes, _ := json.MarshalIndent(info, "", "    ")
    fmt.Printf("Pagination Info:\n%s\n", string(bytes))

    // Expected output:
    // Items: [item1 item2 item3 item4 item5 item6 item7 item8 item9 item10]
    // Pagination Info:
    // {
    //     "current_page": 1,
    //     "total_pages": 10,
    //     "total_items": 100,
    //     "has_next": true,
    //     "has_previous": false,
    //     "items_per_page": 10
    // }
}
```

## Pagination Result Structure

```go
type Pagination struct {
    Items interface{} `json:"items"`
    Pages []int `json:"pages"`
    TotalPages int `json:"total_pages"`
    CurrentPage int `json:"current_page"`
    FirstPage int `json:"first_page"`
    LastPage int `json:"last_page"`
    PreviousPage *int `json:"previous_page"`
    NextPage     *int `json:"next_page"`
    ItemsPerPage int `json:"items_per_page"`
    TotalItems int64 `json:"total_items"`
    FirstPageInRange int `json:"first_page_in_range"`
    LastPageInRange int `json:"last_page_in_range"`
}
```

## Template Usage Example

### HTML Template Example

```html
<!-- pagination.html -->
<div class="pagination">
    {{if .HasPreviousPage}}
        <a href="?page={{.PreviousPage}}">&laquo; Previous</a>
    {{end}}
    {{range .Pages}}
        <a href="?page={{.}}" {{if eq . $.CurrentPage}}class="active"{{end}}>
            {{.}}
        </a>
    {{end}}
    {{if .HasNextPage}}
        <a href="?page={{.NextPage}}">Next &raquo;</a>
    {{end}}
</div>
<div class="info">
    Page {{.CurrentPage}} of {{.TotalPages}}
    (Total: {{.TotalItems}})
</div>
```

### Basic Usage with HTML Handler

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "strconv"
    "html/template"
    
    "github.com/simp-lee/pagination"
)

func handleList(w http.ResponseWriter, r *http.Request) {
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }
    paginator := pagination.NewPaginator(
        pagination.WithItemsPerPage(10),
        pagination.WithPagesInRange(5),
        pagination.WithItemTotalCallback(func(ctx context.Context) (int64, error) {
            return 100, nil // Your total count logic
        }),
        pagination.WithSliceCallback(func(ctx context.Context, offset, limit int) (interface{}, error) {
            return []string{"Item 1", "Item 2"}, nil // Your data fetching logic
        }),
    )

    // Parse template
    tmpl, err := template.ParseFiles("pagination.html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    result, err := paginator.Paginate(r.Context(), page)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    err = tmpl.Execute(w, result)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func main() {
    http.HandleFunc("/", handleList)
    fmt.Println("Server starting on :8080...")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        panic(err)
    }
}
```

## Helper Methods

```go
// Helper Methods for checking pagination state
func (p *Pagination) HasPreviousPage() bool    // Check if there is a previous page
func (p *Pagination) HasNextPage() bool        // Check if there is a next page
func (p *Pagination) IsFirstPage() bool        // Check if current page is the first page
func (p *Pagination) IsLastPage() bool         // Check if current page is the last page
func (p *Pagination) GetPageInfo() map[string]interface{} // Get simplified pagination information
```

## Configuration Options

- `WithItemsPerPage(n int)`: Set the number of items per page
- `WithPagesInRange(n int)`: Set the number of page numbers to show in navigation
- `WithItemTotalCallback(fn func(ctx context.Context) (int64, error))`: Set the callback function to get the total number of items
- `WithSliceCallback(fn func(ctx context.Context, offset, limit int) (interface{}, error))`: Set the callback function to get the slice of items for the current page

## Concurrency Notes

The paginator itself is immutable after creation, but thread safety depends on your callback implementations. Make sure your database operations or other data fetching mechanisms in callbacks are thread-safe if you plan to use the paginator across multiple goroutines.

## License

This project is licensed under the MIT License.

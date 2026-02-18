# Pagination

A type-safe pagination package for Go applications.

## Features

- Generic API (`[]T` end-to-end)
- Context-aware callbacks for count and page slices
- Configurable `items per page` and `pages in range`
- Optional known total (`WithKnownTotal`) to skip count callback
- JSON-ready pagination result with helper methods

## Requirements

- Go 1.25 or later (generics required)

## Installation

```bash
go get github.com/simp-lee/pagination
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/simp-lee/pagination"
)

func main() {
	paginator := pagination.NewPaginator[string](
		pagination.WithItemsPerPage[string](10),
		pagination.WithPagesInRange[string](5),
		pagination.WithItemTotalCallback[string](func(ctx context.Context) (int64, error) {
			return 100, nil
		}),
		pagination.WithSliceCallback[string](func(ctx context.Context, offset, limit int) ([]string, error) {
			return []string{"item1", "item2"}, nil
		}),
	)

	result, err := paginator.Paginate(context.Background(), 1)
	if err != nil {
		panic(err)
	}

	fmt.Println(result.Items)
	fmt.Println(result.TotalPages)
}
```

## Reuse a Known Total (Skip Count Query)

```go
paginator := pagination.NewPaginator[string](
	pagination.WithKnownTotal[string](1000),
	pagination.WithSliceCallback[string](func(ctx context.Context, offset, limit int) ([]string, error) {
		return []string{"item1", "item2"}, nil
	}),
)

result, err := paginator.Paginate(context.Background(), 1)
if err != nil {
	panic(err)
}
fmt.Println(result.TotalItems) // 1000
```

## API

### Sentinel Errors

| Error | Description |
|---|---|
| `ErrInvalidPageNumber` | `currentPage` must be greater than 0 |
| `ErrCallbackNotFound` | A required callback (`sliceCallback` or `itemTotalCallback`/`knownTotal`) is missing |
| `ErrInvalidConfig` | Invalid option value (e.g. `itemsPerPage <= 0`, `pagesInRange <= 0`, `knownTotal < 0`) |

Use `errors.Is` to check:

```go
if errors.Is(err, pagination.ErrInvalidPageNumber) {
	// handle invalid page
}
```

### Types

- `Paginator[T any]`
- `Pagination[T any]`
- `Option[T any]`

### `Pagination[T]` Fields

| Field | Type | JSON Key | Description |
|---|---|---|---|
| `Items` | `[]T` | `items` | Slice of items for the current page |
| `Pages` | `[]int` | `pages` | Page numbers to display in navigation |
| `TotalPages` | `int` | `total_pages` | Total number of pages |
| `CurrentPage` | `int` | `current_page` | Current page number |
| `FirstPage` | `int` | `first_page` | Always `1` |
| `LastPage` | `int` | `last_page` | Equal to `TotalPages` |
| `PreviousPage` | `*int` | `previous_page` | Previous page number; `nil` on first page |
| `NextPage` | `*int` | `next_page` | Next page number; `nil` on last page |
| `ItemsPerPage` | `int` | `items_per_page` | Number of items per page |
| `TotalItems` | `int64` | `total_items` | Total number of items across all pages |
| `FirstPageInRange` | `int` | `first_page_in_range` | First page number in the current navigation range |
| `LastPageInRange` | `int` | `last_page_in_range` | Last page number in the current navigation range |

### Constructors and Options

- `NewPaginator[T](config ...Option[T])`
- `WithItemsPerPage[T](n int)`
- `WithPagesInRange[T](n int)`
- `WithItemTotalCallback[T](cb func(ctx context.Context) (int64, error))`
- `WithSliceCallback[T](cb func(ctx context.Context, offset, limit int) ([]T, error))`
- `WithKnownTotal[T](total int64)`

### Methods

- `(p *Paginator[T]) Paginate(ctx context.Context, currentPage int) (*Pagination[T], error)`
- `(p *Pagination[T]) HasPreviousPage() bool`
- `(p *Pagination[T]) HasNextPage() bool`
- `(p *Pagination[T]) IsFirstPage() bool`
- `(p *Pagination[T]) IsLastPage() bool`

## Configuration Notes

- Invalid values for `WithItemsPerPage` / `WithPagesInRange` are captured and returned by `Paginate`.
- `WithKnownTotal` avoids calling `WithItemTotalCallback` during pagination.
- If `WithKnownTotal` is not set, `WithItemTotalCallback` is required.
- `WithSliceCallback` is always required.

## Behavior Notes

- If `currentPage` is greater than total pages, `Paginate` clamps it to the last page.
- If total items is `0`, pagination still returns `TotalPages = 1` and `CurrentPage = 1`.
- Context cancellation is checked before calling `WithSliceCallback` (after total is resolved).

## License

This project is licensed under the MIT License.

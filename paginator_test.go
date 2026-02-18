package pagination

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestPaginate_EvenPagesInRangeUsesConfiguredWindowSize(t *testing.T) {
	p := NewPaginator[int](
		WithItemsPerPage[int](10),
		WithPagesInRange[int](4),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			return 100, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 5)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}

	want := []int{4, 5, 6, 7}
	if len(result.Pages) != len(want) {
		t.Fatalf("pages len=%d, want=%d, pages=%v", len(result.Pages), len(want), result.Pages)
	}
	for i := range want {
		if result.Pages[i] != want[i] {
			t.Fatalf("pages[%d]=%d, want=%d; pages=%v", i, result.Pages[i], want[i], result.Pages)
		}
	}
}

func TestNewPaginator_InvalidOptionDoesNotPanicAndReturnsErrorOnPaginate(t *testing.T) {
	totalCalled := false
	sliceCalled := false

	p := NewPaginator[int](
		WithItemsPerPage[int](0),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			totalCalled = true
			return 1, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			sliceCalled = true
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid items per page config, got nil")
	}
	if !strings.Contains(err.Error(), "items per page") {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalCalled || sliceCalled {
		t.Fatalf("callbacks should not be called when config is invalid, totalCalled=%v, sliceCalled=%v", totalCalled, sliceCalled)
	}
}

func TestNewPaginator_NilOptionIsIgnored(t *testing.T) {
	p := NewPaginator[int](
		nil,
		WithItemsPerPage[int](10),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			return 1, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil pagination result")
	}
}

func TestPaginate_NegativeTotalReturnsError(t *testing.T) {
	sliceCalled := false
	p := NewPaginator[int](
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			return -1, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			sliceCalled = true
			return []int{}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for negative total items, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if !strings.Contains(err.Error(), "itemTotalCallback failed") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
	if sliceCalled {
		t.Fatal("slice callback should not be called when total is negative")
	}
}

func TestPaginate_WrapsCallbackErrors(t *testing.T) {
	t.Run("item total callback", func(t *testing.T) {
		inner := errors.New("db count failed")
		p := NewPaginator[int](
			WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
				return 0, inner
			}),
			WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
				return nil, nil
			}),
		)

		_, err := p.Paginate(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, inner) {
			t.Fatalf("expected wrapped error to match inner error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "itemTotalCallback failed") {
			t.Fatalf("expected contextual message, got: %v", err)
		}
	})

	t.Run("slice callback", func(t *testing.T) {
		inner := errors.New("db query failed")
		p := NewPaginator[int](
			WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
				return 10, nil
			}),
			WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
				return nil, inner
			}),
		)

		_, err := p.Paginate(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, inner) {
			t.Fatalf("expected wrapped error to match inner error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "sliceCallback failed") {
			t.Fatalf("expected contextual message, got: %v", err)
		}
	})
}

func TestPaginate_EmptyDataAndOutOfRangePageAndNavigationPointers(t *testing.T) {
	p := NewPaginator[int](
		WithItemsPerPage[int](10),
		WithPagesInRange[int](5),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			return 0, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			if offset != 0 {
				t.Fatalf("offset=%d, want=0 for empty dataset", offset)
			}
			if limit != 10 {
				t.Fatalf("limit=%d, want=10", limit)
			}
			return []int{}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 999)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}

	if result.CurrentPage != 1 {
		t.Fatalf("CurrentPage=%d, want=1", result.CurrentPage)
	}
	if result.TotalPages != 1 {
		t.Fatalf("TotalPages=%d, want=1", result.TotalPages)
	}
	if result.PreviousPage != nil {
		t.Fatalf("PreviousPage=%v, want=nil", *result.PreviousPage)
	}
	if result.NextPage != nil {
		t.Fatalf("NextPage=%v, want=nil", *result.NextPage)
	}
}

func TestPaginate_PageCountUsesIntegerMathForLargeTotals(t *testing.T) {
	const total int64 = 9007199254740993 // 2^53 + 1

	p := NewPaginator[int](
		WithItemsPerPage[int](2),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			return total, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}

	wantTotalPages := int(total/2 + 1) // 2^53+1 is odd, so ceiling division adds 1
	if result.TotalPages != wantTotalPages {
		t.Fatalf("TotalPages=%d, want=%d", result.TotalPages, wantTotalPages)
	}
}

func TestPaginate_WithKnownTotalSkipsItemTotalCallback(t *testing.T) {
	totalCalled := false

	p := NewPaginator[int](
		WithKnownTotal[int](25),
		WithItemsPerPage[int](10),
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			totalCalled = true
			return 999, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			if offset != 10 {
				t.Fatalf("offset=%d, want=10", offset)
			}
			if limit != 10 {
				t.Fatalf("limit=%d, want=10", limit)
			}
			return []int{11, 12}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 2)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}
	if totalCalled {
		t.Fatal("itemTotalCallback should not be called when known total is configured")
	}
	if result.TotalItems != 25 {
		t.Fatalf("TotalItems=%d, want=25", result.TotalItems)
	}
}

func TestPaginate_WithKnownTotalDoesNotRequireItemTotalCallback(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](3),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1, 2, 3}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}
	if result.TotalItems != 3 {
		t.Fatalf("TotalItems=%d, want=3", result.TotalItems)
	}
}

func TestNewPaginator_WithKnownTotalRejectsNegative(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](-1),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for negative known total, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestPaginator_TypeSafeStringAPI(t *testing.T) {
	totalCalled := false

	p := NewPaginator[string](
		WithItemsPerPage[string](2),
		WithKnownTotal[string](3),
		WithItemTotalCallback[string](func(ctx context.Context) (int64, error) {
			totalCalled = true
			return 100, nil
		}),
		WithSliceCallback[string](func(ctx context.Context, offset, limit int) ([]string, error) {
			if offset != 0 {
				t.Fatalf("offset=%d, want=0", offset)
			}
			if limit != 2 {
				t.Fatalf("limit=%d, want=2", limit)
			}
			return []string{"a", "b"}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}
	if totalCalled {
		t.Fatal("itemTotalCallback should not be called when known total is configured")
	}
	if len(result.Items) != 2 || result.Items[0] != "a" || result.Items[1] != "b" {
		t.Fatalf("unexpected typed items: %#v", result.Items)
	}
	if result.TotalItems != 3 {
		t.Fatalf("TotalItems=%d, want=3", result.TotalItems)
	}
}

func TestPaginator_RequiresSliceCallback(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](1),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCallbackNotFound) {
		t.Fatalf("expected ErrCallbackNotFound, got: %v", err)
	}
}

func TestPaginator_RequiresTotalSourceWhenKnownTotalMissing(t *testing.T) {
	p := NewPaginator[int](
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCallbackNotFound) {
		t.Fatalf("expected ErrCallbackNotFound, got: %v", err)
	}
}

func TestPaginate_NilItemsSliceBecomesEmptySlice(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](0),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return nil, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}
	if result.Items == nil {
		t.Fatal("Items should not be nil, expected empty slice")
	}
	if len(result.Items) != 0 {
		t.Fatalf("Items len=%d, want=0", len(result.Items))
	}
}

func TestPaginate_MultipleConfigErrorsAccumulated(t *testing.T) {
	p := NewPaginator[int](
		WithItemsPerPage[int](0),
		WithPagesInRange[int](-1),
		WithKnownTotal[int](1),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for multiple invalid configs, got nil")
	}
	if !strings.Contains(err.Error(), "items per page") {
		t.Fatalf("expected 'items per page' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "pages in range") {
		t.Fatalf("expected 'pages in range' in error, got: %v", err)
	}
}

func TestPaginate_ContextCancelledBetweenCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	p := NewPaginator[int](
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			cancel() // cancel after total callback
			return 10, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			t.Fatal("sliceCallback should not be called after context cancellation")
			return nil, nil
		}),
	)

	_, err := p.Paginate(ctx, 1)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestPaginator_InvalidCurrentPageReturnsError(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](1),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidPageNumber) {
		t.Fatalf("expected ErrInvalidPageNumber, got: %v", err)
	}
}

func TestPaginate_CeilingDivisionNoOverflowAtMaxInt64(t *testing.T) {
	total := int64(math.MaxInt64)

	p := NewPaginator[int](
		WithItemsPerPage[int](2),
		WithKnownTotal[int](total),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	result, err := p.Paginate(context.Background(), 1)
	if err != nil {
		t.Fatalf("Paginate returned error: %v", err)
	}

	// math.MaxInt64 is odd, so ceiling division: MaxInt64/2 + 1
	wantTotalPages := int(total/2 + 1)
	if result.TotalPages != wantTotalPages {
		t.Fatalf("TotalPages=%d, want=%d", result.TotalPages, wantTotalPages)
	}
}

func TestPagination_HelperMethods(t *testing.T) {
	p := NewPaginator[int](
		WithItemsPerPage[int](10),
		WithKnownTotal[int](50),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			return []int{1}, nil
		}),
	)

	t.Run("first page", func(t *testing.T) {
		result, err := p.Paginate(context.Background(), 1)
		if err != nil {
			t.Fatalf("Paginate returned error: %v", err)
		}
		if result.HasPreviousPage() {
			t.Fatal("HasPreviousPage() = true, want false on first page")
		}
		if !result.HasNextPage() {
			t.Fatal("HasNextPage() = false, want true on first page with more pages")
		}
		if !result.IsFirstPage() {
			t.Fatal("IsFirstPage() = false, want true")
		}
		if result.IsLastPage() {
			t.Fatal("IsLastPage() = true, want false on first page")
		}
	})

	t.Run("middle page", func(t *testing.T) {
		result, err := p.Paginate(context.Background(), 3)
		if err != nil {
			t.Fatalf("Paginate returned error: %v", err)
		}
		if !result.HasPreviousPage() {
			t.Fatal("HasPreviousPage() = false, want true on middle page")
		}
		if !result.HasNextPage() {
			t.Fatal("HasNextPage() = false, want true on middle page")
		}
		if result.IsFirstPage() {
			t.Fatal("IsFirstPage() = true, want false on middle page")
		}
		if result.IsLastPage() {
			t.Fatal("IsLastPage() = true, want false on middle page")
		}
	})

	t.Run("last page", func(t *testing.T) {
		result, err := p.Paginate(context.Background(), 5)
		if err != nil {
			t.Fatalf("Paginate returned error: %v", err)
		}
		if !result.HasPreviousPage() {
			t.Fatal("HasPreviousPage() = false, want true on last page")
		}
		if result.HasNextPage() {
			t.Fatal("HasNextPage() = true, want false on last page")
		}
		if result.IsFirstPage() {
			t.Fatal("IsFirstPage() = true, want false on last page")
		}
		if !result.IsLastPage() {
			t.Fatal("IsLastPage() = false, want true")
		}
	})
}

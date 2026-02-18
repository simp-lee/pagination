package pagination

import (
	"context"
	"errors"
	"math"
	"strconv"
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
	if !strings.Contains(err.Error(), "total items must not be negative") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
	if sliceCalled {
		t.Fatal("slice callback should not be called when total is negative")
	}
}

func TestPaginator_NilCursorCallback(t *testing.T) {
	p := NewPaginator(WithCursorSliceCallback[int](nil))
	_, err := p.PaginateByCursor(context.TODO(), CursorRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error for nil cursorSliceCallback, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestPaginator_NilKeysetCallback(t *testing.T) {
	p := NewPaginator(WithKeysetSliceCallback[int](nil))
	_, err := p.PaginateByKeyset(context.TODO(), KeysetRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error for nil keysetSliceCallback, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
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

func TestPaginate_ContextAlreadyCancelledSkipsItemTotalCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	totalCalled := false
	p := NewPaginator[int](
		WithItemTotalCallback[int](func(ctx context.Context) (int64, error) {
			totalCalled = true
			return 10, nil
		}),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			t.Fatal("sliceCallback should not be called when context is already canceled")
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
	if totalCalled {
		t.Fatal("itemTotalCallback should not be called when context is already canceled")
	}
}

func TestPaginate_NilContextReturnsError(t *testing.T) {
	p := NewPaginator[int](
		WithKnownTotal[int](1),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			t.Fatal("sliceCallback should not be called when context is nil")
			return nil, nil
		}),
	)

	_, err := p.Paginate(context.TODO(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if !strings.Contains(err.Error(), "context must not be nil") {
		t.Fatalf("expected nil-context error message, got: %v", err)
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

func TestPaginate_PageCountOverflowReturnsErrorOn32Bit(t *testing.T) {
	if strconv.IntSize != 32 {
		t.Skip("architecture is not 32-bit")
	}

	sliceCalled := false
	p := NewPaginator[int](
		WithItemsPerPage[int](1),
		WithKnownTotal[int](math.MaxInt64),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			sliceCalled = true
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), 1)
	if err == nil {
		t.Fatal("expected overflow protection error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if sliceCalled {
		t.Fatal("slice callback should not be called when page count overflows int")
	}
}

func TestPaginate_OffsetOverflowReturnsErrorOn32Bit(t *testing.T) {
	if strconv.IntSize != 32 {
		t.Skip("architecture is not 32-bit")
	}

	maxInt := int(^uint(0) >> 1)
	total := int64(maxInt) * 2
	sliceCalled := false

	p := NewPaginator[int](
		WithItemsPerPage[int](2),
		WithKnownTotal[int](total),
		WithSliceCallback[int](func(ctx context.Context, offset, limit int) ([]int, error) {
			sliceCalled = true
			return []int{1}, nil
		}),
	)

	_, err := p.Paginate(context.Background(), maxInt)
	if err == nil {
		t.Fatal("expected overflow protection error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if sliceCalled {
		t.Fatal("slice callback should not be called when offset overflows int")
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

func TestPaginateByCursor_ForwardAndBoundary(t *testing.T) {
	after := "cursor:10"
	next := "cursor:20"
	prev := "cursor:0"

	p := NewPaginator[string](
		WithItemsPerPage[string](2),
		WithCursorSliceCallback[string](func(ctx context.Context, req CursorRequest) (*CursorResult[string], error) {
			if req.AfterCursor == nil || *req.AfterCursor != after {
				t.Fatalf("after cursor=%v, want=%q", req.AfterCursor, after)
			}
			if req.Limit != 2 {
				t.Fatalf("limit=%d, want=2", req.Limit)
			}
			if req.Direction != DirectionForward {
				t.Fatalf("direction=%q, want=%q", req.Direction, DirectionForward)
			}
			return &CursorResult[string]{
				Items:          []string{"a", "b"},
				NextCursor:     &next,
				PreviousCursor: &prev,
				HasMore:        true,
			}, nil
		}),
	)

	result, err := p.PaginateByCursor(context.Background(), CursorRequest{AfterCursor: &after})
	if err != nil {
		t.Fatalf("PaginateByCursor returned error: %v", err)
	}
	if len(result.Items) != 2 || result.Items[0] != "a" || result.Items[1] != "b" {
		t.Fatalf("unexpected items: %#v", result.Items)
	}
	if result.NextCursor == nil || *result.NextCursor != next {
		t.Fatalf("NextCursor=%v, want=%q", result.NextCursor, next)
	}
	if result.PreviousCursor == nil || *result.PreviousCursor != prev {
		t.Fatalf("PreviousCursor=%v, want=%q", result.PreviousCursor, prev)
	}
	if !result.HasMore {
		t.Fatal("HasMore=false, want=true")
	}
	if !result.HasNext() {
		t.Fatal("HasNext()=false, want=true")
	}
	if !result.HasPrevious() {
		t.Fatal("HasPrevious()=false, want=true")
	}

	pBoundary := NewPaginator[string](
		WithCursorSliceCallback[string](func(ctx context.Context, req CursorRequest) (*CursorResult[string], error) {
			return &CursorResult[string]{
				Items:   []string{},
				HasMore: false,
			}, nil
		}),
	)

	boundary, err := pBoundary.PaginateByCursor(context.Background(), CursorRequest{Limit: 1})
	if err != nil {
		t.Fatalf("PaginateByCursor returned error: %v", err)
	}
	if boundary.HasMore {
		t.Fatal("HasMore=true, want=false")
	}
	if boundary.HasNext() {
		t.Fatal("HasNext()=true, want=false")
	}
}

func TestPaginateByCursor_InvalidRequestAndErrors(t *testing.T) {
	after := "a"
	before := "b"

	p := NewPaginator[int](
		WithCursorSliceCallback[int](func(ctx context.Context, req CursorRequest) (*CursorResult[int], error) {
			return &CursorResult[int]{Items: []int{1}}, nil
		}),
	)

	_, err := p.PaginateByCursor(context.Background(), CursorRequest{AfterCursor: &after, BeforeCursor: &before})
	if err == nil {
		t.Fatal("expected error for invalid cursor request, got nil")
	}
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got: %v", err)
	}

	inner := errors.New("query failed")
	pWrap := NewPaginator[int](
		WithCursorSliceCallback[int](func(ctx context.Context, req CursorRequest) (*CursorResult[int], error) {
			return nil, inner
		}),
	)

	_, err = pWrap.PaginateByCursor(context.Background(), CursorRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected wrapped callback error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected wrapped inner error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "cursorSliceCallback failed") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestPaginateByCursor_RequiresCallbackAndContextCancel(t *testing.T) {
	p := NewPaginator[int]()
	_, err := p.PaginateByCursor(context.Background(), CursorRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCallbackNotFound) {
		t.Fatalf("expected ErrCallbackNotFound, got: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p2 := NewPaginator[int](
		WithCursorSliceCallback[int](func(ctx context.Context, req CursorRequest) (*CursorResult[int], error) {
			t.Fatal("cursor callback should not be called for cancelled context")
			return nil, nil
		}),
	)
	_, err = p2.PaginateByCursor(ctx, CursorRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestPaginateByCursor_NilContextReturnsError(t *testing.T) {
	p := NewPaginator[int](
		WithCursorSliceCallback[int](func(ctx context.Context, req CursorRequest) (*CursorResult[int], error) {
			t.Fatal("cursor callback should not be called when context is nil")
			return nil, nil
		}),
	)

	_, err := p.PaginateByCursor(context.TODO(), CursorRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if !strings.Contains(err.Error(), "context must not be nil") {
		t.Fatalf("expected nil-context error message, got: %v", err)
	}
}

func TestPaginateByKeyset_BackwardAndValidation(t *testing.T) {
	before := "2026-01-10T12:00:00Z|42"
	next := "2026-01-10T12:00:00Z|41"
	prev := "2026-01-10T12:00:00Z|43"

	p := NewPaginator[string](
		WithItemsPerPage[string](3),
		WithKeysetSliceCallback[string](func(ctx context.Context, req KeysetRequest) (*KeysetResult[string], error) {
			if req.BeforeKey == nil || *req.BeforeKey != before {
				t.Fatalf("before key=%v, want=%q", req.BeforeKey, before)
			}
			if req.AfterKey != nil {
				t.Fatalf("after key=%v, want=nil", req.AfterKey)
			}
			if req.Limit != 3 {
				t.Fatalf("limit=%d, want=3", req.Limit)
			}
			if req.Direction != DirectionBackward {
				t.Fatalf("direction=%q, want=%q", req.Direction, DirectionBackward)
			}
			return &KeysetResult[string]{
				Items:       []string{"r43", "r42", "r41"},
				NextKey:     &next,
				PreviousKey: &prev,
				HasMore:     true,
			}, nil
		}),
	)

	result, err := p.PaginateByKeyset(context.Background(), KeysetRequest{
		BeforeKey: &before,
		Direction: DirectionBackward,
	})
	if err != nil {
		t.Fatalf("PaginateByKeyset returned error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("items len=%d, want=3", len(result.Items))
	}
	if result.NextKey == nil || *result.NextKey != next {
		t.Fatalf("NextKey=%v, want=%q", result.NextKey, next)
	}
	if result.PreviousKey == nil || *result.PreviousKey != prev {
		t.Fatalf("PreviousKey=%v, want=%q", result.PreviousKey, prev)
	}
	if !result.HasNext() || !result.HasPrevious() {
		t.Fatalf("expected next/previous keys to be available: %+v", result)
	}

	after := "k1"
	_, err = p.PaginateByKeyset(context.Background(), KeysetRequest{AfterKey: &after, BeforeKey: &before})
	if err == nil {
		t.Fatal("expected keyset validation error, got nil")
	}
	if !errors.Is(err, ErrInvalidKeyset) {
		t.Fatalf("expected ErrInvalidKeyset, got: %v", err)
	}
}

func TestPaginateByKeyset_NilContextReturnsError(t *testing.T) {
	p := NewPaginator[int](
		WithKeysetSliceCallback[int](func(ctx context.Context, req KeysetRequest) (*KeysetResult[int], error) {
			t.Fatal("keyset callback should not be called when context is nil")
			return nil, nil
		}),
	)

	_, err := p.PaginateByKeyset(context.TODO(), KeysetRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
	if !strings.Contains(err.Error(), "context must not be nil") {
		t.Fatalf("expected nil-context error message, got: %v", err)
	}
}

// --- Minor 1: Keyset pagination test parity with cursor pagination ---

func TestPaginateByKeyset_CallbackError(t *testing.T) {
	inner := errors.New("keyset query failed")
	p := NewPaginator[int](
		WithKeysetSliceCallback[int](func(ctx context.Context, req KeysetRequest) (*KeysetResult[int], error) {
			return nil, inner
		}),
	)

	_, err := p.PaginateByKeyset(context.Background(), KeysetRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected wrapped callback error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected wrapped inner error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "keysetSliceCallback failed") {
		t.Fatalf("expected contextual message, got: %v", err)
	}
}

func TestPaginateByKeyset_RequiresCallback(t *testing.T) {
	p := NewPaginator[int]()
	_, err := p.PaginateByKeyset(context.Background(), KeysetRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCallbackNotFound) {
		t.Fatalf("expected ErrCallbackNotFound, got: %v", err)
	}
}

func TestPaginateByKeyset_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := NewPaginator[int](
		WithKeysetSliceCallback[int](func(ctx context.Context, req KeysetRequest) (*KeysetResult[int], error) {
			t.Fatal("keyset callback should not be called for cancelled context")
			return nil, nil
		}),
	)

	_, err := p.PaginateByKeyset(ctx, KeysetRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

// --- Minor 2: Invalid direction and negative limit tests ---

func TestPaginateByCursor_InvalidDirectionAndNegativeLimit(t *testing.T) {
	p := NewPaginator[int](
		WithCursorSliceCallback[int](func(ctx context.Context, req CursorRequest) (*CursorResult[int], error) {
			t.Fatal("callback should not be called for invalid request")
			return nil, nil
		}),
	)

	t.Run("invalid direction", func(t *testing.T) {
		_, err := p.PaginateByCursor(context.Background(), CursorRequest{
			Limit:     1,
			Direction: "invalid",
		})
		if err == nil {
			t.Fatal("expected error for invalid direction, got nil")
		}
		if !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("expected ErrInvalidCursor, got: %v", err)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		_, err := p.PaginateByCursor(context.Background(), CursorRequest{
			Limit: -1,
		})
		if err == nil {
			t.Fatal("expected error for negative limit, got nil")
		}
		if !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("expected ErrInvalidCursor, got: %v", err)
		}
	})
}

func TestPaginateByKeyset_InvalidDirectionAndNegativeLimit(t *testing.T) {
	p := NewPaginator[int](
		WithKeysetSliceCallback[int](func(ctx context.Context, req KeysetRequest) (*KeysetResult[int], error) {
			t.Fatal("callback should not be called for invalid request")
			return nil, nil
		}),
	)

	t.Run("invalid direction", func(t *testing.T) {
		_, err := p.PaginateByKeyset(context.Background(), KeysetRequest{
			Limit:     1,
			Direction: "invalid",
		})
		if err == nil {
			t.Fatal("expected error for invalid direction, got nil")
		}
		if !errors.Is(err, ErrInvalidKeyset) {
			t.Fatalf("expected ErrInvalidKeyset, got: %v", err)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		_, err := p.PaginateByKeyset(context.Background(), KeysetRequest{
			Limit: -1,
		})
		if err == nil {
			t.Fatal("expected error for negative limit, got nil")
		}
		if !errors.Is(err, ErrInvalidKeyset) {
			t.Fatalf("expected ErrInvalidKeyset, got: %v", err)
		}
	})
}

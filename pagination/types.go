package pagination

import (
	"context"
	"errors"
	"reflect"
	"time"
)

// Arguments represents pagination arguments.
// The arguments can be used to paginate forward or backward.
//
// To enable forward pagination, two arguments must be provided:
//   - first: the number of items to retrieve
//   - after: the cursor of the last item of the previous page
//
// The server will return at most `first` items after the  `after` cursor.
//
// To enable backward pagination, two arguments must be provided:
//   - last: the number of items to retrieve
//   - before: the cursor of the first item of the previous page
//
// The server will return at most `last` items before the  `before` cursor.
//
// The items may be ordered however the business logic dictates. The ordering
// may be determined based upon additional arguments
// given to each implementation.
type Arguments struct {
	First *int
	After *string

	Last   *int
	Before *string

	afterCursor  *Cursor
	beforeCursor *Cursor
}

// PageInfo represents pagination information.
type PageInfo struct {
	// HasPreviousPage is used to indicate whether more items exist prior to the
	// set defined by the pagination arguments.
	// If the client is paginating with last/before,
	// then the server must return true if prior items exist, otherwise false.
	// If the client is paginating with first/after, then the client may return true if
	// items prior to after exist, if it can do so efficiently, otherwise may return false.
	HasPreviousPage bool
	// HasNextPage is used to indicate whether more items exist following the
	// set defined by the pagination arguments.
	// If the client is paginating with first/after, then the server must return true
	// if further items exist, otherwise false.
	// If the client is paginating with last/before, then the server may return true if
	// items further from before exist, if it can do so efficiently, otherwise may return false.
	HasNextPage bool

	// StartCursor corresponds to the first item in the result set.
	StartCursor *string
	// EndCursor corresponds to the last item in the result set.
	EndCursor *string
}

// PaginatedItem contains a generic item and its cursor.
//
// In order to be able to compute the Cursor,
// the expectation is that T defines a field named ID of type string and
// field named CreatedAt of type time.Time or *time.Time.
type PaginatedItem[T any] struct {
	Item   T
	Cursor string
}

var (
	// ErrCursorFieldNotFound is returned when a cursor field is not found.
	ErrCursorFieldNotFound = errors.New("field not found")
	// ErrCursorInvalidValueType is returned when a cursor field is not of the correct type.
	ErrCursorInvalidValueType = errors.New("invalid value type")
)

var timeKind = reflect.TypeOf(time.Time{}).Kind()

// Paginator defines methods for retrieving paginated items.
// The items may be ordered however the business logic dictates.
// The ordering must be consistent from page to page.
//
// !
// The ordering of items should be the same when using first/after as when
// using last/before, all other arguments being equal.
// It should not be reversed when using last/before.
// !
//
// When before: cursor is used, the edge closest to cursor must come last in the result edges.
// When after: cursor is used, the edge closest to cursor must come first in the result edges.
type Paginator[T any] interface {
	ListItems(ctx context.Context, pagination Arguments) ([]PaginatedItem[T], PageInfo, error)
}

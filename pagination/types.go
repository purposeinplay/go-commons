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
	HasPreviousPage bool
	HasNextPage     bool

	StartCursor *string
	EndCursor   *string
}

// PaginatedItem contains a generic item and its cursor.
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

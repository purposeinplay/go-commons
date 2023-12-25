package pagination

import (
	"reflect"
	"errors"
	"time"
	"context"
)

type Params struct {
	First *int
	After *string

	Last   *int
	Before *string

	afterCursor  *Cursor
	beforeCursor *Cursor
}

type PageInfo struct {
	HasPreviousPage bool
	HasNextPage     bool
}

type PaginatedItem[T any] struct {
	Item   T
	Cursor string
}

var (
	ErrFieldNotFound    = errors.New("field not found")
	ErrInvalidValueType = errors.New("invalid value type")
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
	ListItems(ctx context.Context, pagination Params) ([]PaginatedItem[T], PageInfo, error)
}

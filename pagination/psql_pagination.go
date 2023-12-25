package pagination

import (
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"
)

var _ Paginator[any] = (*PSQLPaginator[any])(nil)

// PSQLPaginator implements the Paginator interface for
// PostgreSQL databases.
type PSQLPaginator[T any] struct {
	DB *gorm.DB
}

// ListItems retrieves the paginated result set.
// nolint: gocognit, gocyclo
func (p PSQLPaginator[T]) ListItems(
	_ context.Context,
	paginationParams Arguments,
) ([]PaginatedItem[T], PageInfo, error) {
	var err error

	if paginationParams.After != nil {
		paginationParams.afterCursor, err = (&Cursor{}).SetString(*paginationParams.After)
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("decode after cursor: %w", err)
		}
	}

	if paginationParams.Before != nil {
		paginationParams.beforeCursor, err = (&Cursor{}).SetString(*paginationParams.Before)
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("decode before cursor: %w", err)
		}
	}

	items, err := queryItems[T](p.DB, paginationParams)
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("query items: %w", err)
	}

	var model T

	pageInfoSession := p.DB.Session(&gorm.Session{}).Model(model)

	pageInfo, err := getPageInfo(pageInfoSession, int64(len(items)), paginationParams)
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("get page info: %w", err)
	}

	paginatedItems := make([]PaginatedItem[T], len(items))

	for i := range items {
		cursor, err := computeItemCursor(items[i])
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("compute cursor: %w", err)
		}

		paginatedItems[i] = PaginatedItem[T]{
			Item:   items[i],
			Cursor: cursor,
		}
	}

	if len(paginatedItems) > 0 {
		pageInfo.StartCursor = &paginatedItems[0].Cursor
		pageInfo.EndCursor = &paginatedItems[len(paginatedItems)-1].Cursor
	}

	return paginatedItems, pageInfo, nil
}

func queryItems[T any](ses *gorm.DB, pagination Arguments) ([]T, error) {
	var items []T

	pagSession := ses.Session(&gorm.Session{})

	// First/After
	if pagination.First != nil {
		pagSession = pagSession.Order("created_at DESC").Limit(*pagination.First)

		if pagination.afterCursor != nil {
			pagSession = pagSession.Where("created_at < ?", pagination.afterCursor.CreatedAt)
		}

		if err := pagSession.Find(&items).Error; err != nil {
			return nil, fmt.Errorf("find items: %w", err)
		}
	}

	// Last/Before
	if pagination.Last != nil {
		pagSession = pagSession.Order("created_at ASC").Limit(*pagination.Last)

		if pagination.beforeCursor != nil {
			pagSession = pagSession.Where("created_at > ?", pagination.beforeCursor.CreatedAt)
		}

		if err := pagSession.Find(&items).Error; err != nil {
			return nil, fmt.Errorf("find items: %w", err)
		}

		slices.Reverse(items)
	}

	return items, nil
}

func getPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination Arguments,
) (PageInfo, error) {
	if pagination.First != nil {
		return getForwardPaginationPageInfo(db, itemsCount, pagination)
	}

	return getBackwardPaginationPageInfo(db, itemsCount, pagination)
}

func getForwardPaginationPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination Arguments,
) (PageInfo, error) {
	var (
		totalItemsForward  int64
		totalItemsBackward int64
	)

	totalItemsForwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	totalItemsBackwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	if pagination.afterCursor == nil {
		if err := totalItemsForwardQuery.Count(&totalItemsForward).Error; err != nil {
			return PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
		}

		return PageInfo{
			HasPreviousPage: false,
			HasNextPage:     totalItemsForward > itemsCount,
		}, nil
	}

	totalItemsForwardQuery = totalItemsForwardQuery.Where(
		"created_at < ?",
		pagination.afterCursor.CreatedAt,
	)

	totalItemsBackwardQuery = totalItemsBackwardQuery.Debug().Where(
		"created_at > ?",
		pagination.afterCursor.CreatedAt,
	)

	if err := totalItemsForwardQuery.Count(&totalItemsForward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	if err := totalItemsBackwardQuery.Count(&totalItemsBackward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for backward pagination: %w", err)
	}

	return PageInfo{
		HasPreviousPage: totalItemsBackward > 0,
		HasNextPage:     totalItemsForward > itemsCount,
	}, nil
}

func getBackwardPaginationPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination Arguments,
) (PageInfo, error) {
	var (
		totalItemsForward  int64
		totalItemsBackward int64
	)

	totalItemsForwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	totalItemsBackwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	if pagination.Before == nil {
		if err := totalItemsBackwardQuery.Count(&totalItemsBackward).Error; err != nil {
			return PageInfo{}, fmt.Errorf("count items for backward pagination: %w", err)
		}

		return PageInfo{
			HasPreviousPage: totalItemsBackward > itemsCount,
			HasNextPage:     false,
		}, nil
	}

	totalItemsForwardQuery = totalItemsForwardQuery.Where(
		"created_at < ?",
		pagination.beforeCursor.CreatedAt,
	)

	totalItemsBackwardQuery = totalItemsBackwardQuery.Where(
		"created_at > ?",
		pagination.beforeCursor.CreatedAt,
	)

	if err := totalItemsForwardQuery.Count(&totalItemsForward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	if err := totalItemsBackwardQuery.Count(&totalItemsBackward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for backward pagination: %w", err)
	}

	return PageInfo{
		HasPreviousPage: totalItemsBackward > itemsCount,
		HasNextPage:     totalItemsForward > 0,
	}, nil
}

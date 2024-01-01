package pagination

import (
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"
	"k8s.io/utils/ptr"
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

	var (
		startCursor *Cursor
		endCursor   *Cursor
	)

	if len(paginatedItems) > 0 {
		startCursor, err = (&Cursor{}).SetString(paginatedItems[0].Cursor)
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("decode start cursor: %w", err)
		}

		endCursor, err = (&Cursor{}).SetString(paginatedItems[len(paginatedItems)-1].Cursor)
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("decode end cursor: %w", err)
		}
	}

	pageInfo, err := getPageInfo(
		pageInfoSession,
		paginationParams,
		startCursor,
		endCursor,
	)
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("get page info: %w", err)
	}

	if len(paginatedItems) > 0 {
		pageInfo.StartCursor = ptr.To(startCursor.String())
		pageInfo.EndCursor = ptr.To(endCursor.String())
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
	pagination Arguments,
	startCursor *Cursor,
	endCursor *Cursor,
) (PageInfo, error) {
	if pagination.First != nil {
		return getForwardPaginationPageInfo(db, pagination, endCursor)
	}

	return getBackwardPaginationPageInfo(db, pagination, startCursor)
}

func getForwardPaginationPageInfo(
	db *gorm.DB,
	pagination Arguments,
	endCursor *Cursor,
) (PageInfo, error) {
	var (
		hasItemForward  int64
		hasItemBackward int64
	)

	itemForwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	itemBackwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	var pageInfo PageInfo

	if endCursor != nil {
		itemForwardQuery = itemForwardQuery.Where(
			"created_at < ?",
			endCursor.CreatedAt,
		)
	} else if pagination.afterCursor != nil {
		// Case where zero items are fetched but with a cursor.
		itemForwardQuery = itemForwardQuery.Where(
			"created_at < ?",
			pagination.afterCursor.CreatedAt,
		)
	}

	if err := itemForwardQuery.Count(&hasItemForward).
		Limit(1).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	pageInfo.HasNextPage = hasItemForward > 0

	if pagination.afterCursor == nil {
		pageInfo.HasPreviousPage = false
		return pageInfo, nil
	}

	itemBackwardQuery = itemBackwardQuery.Where(
		"created_at > ?",
		pagination.afterCursor.CreatedAt,
	)

	if err := itemBackwardQuery.Count(&hasItemBackward).
		Limit(1).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for backward pagination: %w", err)
	}

	pageInfo.HasPreviousPage = hasItemBackward > 0

	return pageInfo, nil
}

func getBackwardPaginationPageInfo(
	db *gorm.DB,
	pagination Arguments,
	startCursor *Cursor,
) (PageInfo, error) {
	var (
		hasItemForward  int64
		hasItemBackward int64
	)

	itemForwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	itemBackwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	var pageInfo PageInfo

	if startCursor != nil {
		itemBackwardQuery = itemBackwardQuery.Where(
			"created_at > ?",
			startCursor.CreatedAt,
		)
	} else if pagination.beforeCursor != nil {
		// Case where zero items are fetched but with a cursor.
		itemBackwardQuery = itemBackwardQuery.Where(
			"created_at > ?",
			pagination.beforeCursor.CreatedAt,
		)
	}

	if err := itemBackwardQuery.Count(&hasItemBackward).
		Limit(1).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for backward pagination: %w", err)
	}

	pageInfo.HasPreviousPage = hasItemBackward > 0

	if pagination.beforeCursor == nil {
		pageInfo.HasNextPage = false
		return pageInfo, nil
	}

	itemForwardQuery = itemForwardQuery.Where(
		"created_at < ?",
		pagination.beforeCursor.CreatedAt,
	)

	if err := itemForwardQuery.Count(&hasItemForward).
		Limit(1).Error; err != nil {
		return PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	pageInfo.HasNextPage = hasItemForward > 0

	return pageInfo, nil
}

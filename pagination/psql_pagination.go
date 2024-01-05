package pagination

import (
	"context"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"k8s.io/utils/ptr"
)

var _ Paginator[Tabler] = (*PSQLPaginator[Tabler])(nil)

// PSQLPaginator implements the Paginator interface for
// PostgreSQL databases.
type PSQLPaginator[T Tabler] struct {
	DB *gorm.DB
}

// ListItems retrieves the paginated result set.
// nolint: gocognit, gocyclo
func (p PSQLPaginator[T]) ListItems(
	_ context.Context,
	paginationParams Arguments,
) (*Page[T], error) {
	var err error

	if paginationParams.After != nil {
		paginationParams.afterCursor, err = (&Cursor{}).SetString(*paginationParams.After)
		if err != nil {
			return nil, fmt.Errorf("decode after cursor: %w", err)
		}
	}

	if paginationParams.Before != nil {
		paginationParams.beforeCursor, err = (&Cursor{}).SetString(*paginationParams.Before)
		if err != nil {
			return nil, fmt.Errorf("decode before cursor: %w", err)
		}
	}

	items, err := queryItems[T](p.DB, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}

	var model T

	pageInfoSession := p.DB.Session(&gorm.Session{}).Model(model)

	var (
		paginatedItems = make([]PaginatedItem[T], len(items))
		startCursor    *Cursor
		endCursor      *Cursor
	)

	for i := range items {
		cursor, err := computeItemCursor(items[i])
		if err != nil {
			return nil, fmt.Errorf("compute cursor: %w", err)
		}

		if i == 0 {
			startCursor = &cursor
		}

		if i == len(items)-1 {
			endCursor = &cursor
		}

		paginatedItems[i] = PaginatedItem[T]{
			Item:   items[i],
			Cursor: cursor.String(),
		}
	}

	pageInfo, err := getPageInfo[T](
		pageInfoSession,
		paginationParams,
		startCursor,
		endCursor,
	)
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	if len(paginatedItems) > 0 {
		pageInfo.StartCursor = ptr.To(paginatedItems[0].Cursor)
		pageInfo.EndCursor = ptr.To(paginatedItems[len(paginatedItems)-1].Cursor)
	}

	return &Page[T]{
		Items: paginatedItems,
		Info:  pageInfo,
	}, nil
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

func getPageInfo[T Tabler](
	db *gorm.DB,
	pagination Arguments,
	startCursor *Cursor,
	endCursor *Cursor,
) (PageInfo, error) {
	if pagination.First != nil {
		return getForwardPaginationPageInfo[T](db, pagination, endCursor)
	}

	return getBackwardPaginationPageInfo[T](db, pagination, startCursor)
}

func getForwardPaginationPageInfo[T Tabler](
	db *gorm.DB,
	pagination Arguments,
	endCursor *Cursor,
) (PageInfo, error) {
	var (
		hasItemForward  bool
		hasItemBackward bool
	)

	var pageInfo PageInfo

	createdAt := time.Now()

	if endCursor != nil {
		createdAt = endCursor.CreatedAt
	} else if pagination.afterCursor != nil {
		// Case where zero items are fetched but with a cursor.
		createdAt = pagination.afterCursor.CreatedAt
	}

	var model T
	tableName := model.TableName()

	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE created_at < ?)", tableName)

	if err := db.Raw(query, createdAt).Take(&hasItemForward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("existence check for forward pagination: %w", err)
	}

	pageInfo.HasNextPage = hasItemForward

	if pagination.afterCursor == nil {
		pageInfo.HasPreviousPage = false
		return pageInfo, nil
	}

	query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE created_at > ?)", tableName)

	if err := db.Raw(
		query,
		pagination.afterCursor.CreatedAt,
	).Take(&hasItemBackward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("existence check for backward pagination: %w", err)
	}

	pageInfo.HasPreviousPage = hasItemBackward

	return pageInfo, nil
}

func getBackwardPaginationPageInfo[T Tabler](
	db *gorm.DB,
	pagination Arguments,
	startCursor *Cursor,
) (PageInfo, error) {
	var (
		hasItemForward  bool
		hasItemBackward bool
	)

	var pageInfo PageInfo

	var createdAt time.Time

	if startCursor != nil {
		createdAt = startCursor.CreatedAt
	} else if pagination.beforeCursor != nil {
		// Case where zero items are fetched but with a cursor.
		createdAt = pagination.beforeCursor.CreatedAt
	}

	var model T
	tableName := model.TableName()

	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE created_at > ?)", tableName)
	if err := db.Raw(query, createdAt).Take(&hasItemBackward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("existence check for backward pagination: %w", err)
	}

	pageInfo.HasPreviousPage = hasItemBackward

	if pagination.beforeCursor == nil {
		pageInfo.HasNextPage = false
		return pageInfo, nil
	}

	query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE created_at < ?)", tableName)
	if err := db.Raw(query, pagination.beforeCursor.CreatedAt).
		Take(&hasItemForward).Error; err != nil {
		return PageInfo{}, fmt.Errorf("existence check for forward pagination: %w", err)
	}

	pageInfo.HasNextPage = hasItemForward

	return pageInfo, nil
}

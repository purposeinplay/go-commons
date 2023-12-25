package paginationpsql

import (
	"gorm.io/gorm"
	"fmt"
	"github.com/purposeinplay/go-commons/pagination"
)

func ListPSQLPaginatedItems[T any](
	db *gorm.DB,
	pagination pagination.Params,
) ([]pagination.PaginatedItem[T], pagination.PageInfo, error) {
	var err error

	if pagination.After != nil {
		pagination.afterCursor, err = (&pagination.Cursor{}).SetString(*pagination.After)
		if err != nil {
			return nil, pagination.PageInfo{}, fmt.Errorf("decode after cursor: %w", err)
		}
	}

	if pagination.Before != nil {
		pagination.beforeCursor, err = (&pagination.Cursor{}).SetString(*pagination.Before)
		if err != nil {
			return nil, pagination.PageInfo{}, fmt.Errorf("decode before cursor: %w", err)
		}
	}

	paginationQuery, err := paginationToQuery(db, pagination)
	if err != nil {
		return nil, pagination.PageInfo{}, fmt.Errorf("pagination to query: %w", err)
	}

	var items []T

	if err := paginationQuery.Session(&gorm.Session{}).Find(&items).Error; err != nil {
		return nil, pagination.PageInfo{}, fmt.Errorf("find items: %w", err)
	}

	var model T

	db = db.Model(model)

	pageInfo, err := getPageInfo(db.Session(&gorm.Session{}), int64(len(items)), pagination)
	if err != nil {
		return nil, pagination.PageInfo{}, fmt.Errorf("get page info: %w", err)
	}

	paginatedItems := make([]pagination.PaginatedItem[T], len(items))

	for i := range items {
		cursor, err := pagination.computeStructCursor(items[i])
		if err != nil {
			return nil, pagination.PageInfo{}, fmt.Errorf("compute cursor: %w", err)
		}

		paginatedItems[i] = pagination.PaginatedItem[T]{
			Item:   items[i],
			Cursor: cursor,
		}
	}

	return paginatedItems, pageInfo, nil
}

// nolint:revive
func paginationToQuery(ses *gorm.DB, pagination pagination.Params) (*gorm.DB, error) {
	// First/After
	if pagination.First != nil {
		ses = ses.Limit(*pagination.First)
	}

	if pagination.afterCursor != nil {
		ses = ses.Where("created_at > ?", pagination.afterCursor.createdAt)
	}

	// Last/Before
	if pagination.Last != nil {
		ses = ses.Limit(*pagination.First)
	}

	if pagination.beforeCursor != nil {
		ses = ses.Where("created_at < ?", pagination.beforeCursor.createdAt)
	}

	return ses, nil
}

func getPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination pagination.Params,
) (pagination.PageInfo, error) {
	if pagination.First != nil {
		return getForwardPaginationPageInfo(db, itemsCount, pagination)
	}

	return getBackwardsPaginationPageInfo(db, itemsCount, pagination)
}

func getForwardPaginationPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination pagination.Params,
) (pagination.PageInfo, error) {
	var (
		totalItemsForward   int64
		totalItemsBackwards int64
	)

	defer func() {
		fmt.Println(totalItemsForward, totalItemsBackwards, itemsCount)
	}()

	totalItemsForwardQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	totalItemsBackwardsQuery := db.
		Session(&gorm.Session{}).
		Order("created_at DESC")

	if pagination.afterCursor == nil {
		if err := totalItemsForwardQuery.Count(&totalItemsForward).Error; err != nil {
			return pagination.PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
		}

		return pagination.PageInfo{
			HasPreviousPage: false,
			HasNextPage:     totalItemsForward > itemsCount,
		}, nil
	}

	totalItemsForwardQuery = totalItemsForwardQuery.Where(
		"created_at > ?",
		pagination.afterCursor.createdAt,
	)

	totalItemsBackwardsQuery = totalItemsBackwardsQuery.Debug().Where(
		"created_at < ?",
		pagination.afterCursor.createdAt,
	)

	if err := totalItemsForwardQuery.Count(&totalItemsForward).Error; err != nil {
		return pagination.PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	if err := totalItemsBackwardsQuery.Count(&totalItemsBackwards).Error; err != nil {
		return pagination.PageInfo{}, fmt.Errorf("count items for backwards pagination: %w", err)
	}

	return pagination.PageInfo{
		HasPreviousPage: totalItemsBackwards > 0,
		HasNextPage:     totalItemsForward > itemsCount,
	}, nil
}

func getBackwardsPaginationPageInfo(
	db *gorm.DB,
	itemsCount int64,
	pagination pagination.Params,
) (pagination.PageInfo, error) {
	var (
		totalItemsForward   int64
		totalItemsBackwards int64
	)

	totalItemsForwardQuery := db.
		Order("created_at ASC").
		Count(&totalItemsForward)

	totalItemsBackwardsQuery := db.
		Order("created_at ASC").
		Count(&totalItemsBackwards)

	if pagination.Before == nil {
		if err := totalItemsBackwardsQuery.Error; err != nil {
			return pagination.PageInfo{}, fmt.Errorf("count items for backwards pagination: %w", err)
		}

		return pagination.PageInfo{
			HasPreviousPage: totalItemsBackwards > itemsCount,
			HasNextPage:     false,
		}, nil
	}

	cursor, err := (&pagination.Cursor{}).SetString(*pagination.Before)
	if err != nil {
		return pagination.PageInfo{}, fmt.Errorf("decode cursor before: %w", err)
	}

	totalItemsForwardQuery = totalItemsForwardQuery.Where(
		"created_at > ?",
		cursor.createdAt,
	)

	totalItemsBackwardsQuery = totalItemsBackwardsQuery.Where(
		"created_at < ?",
		cursor.createdAt,
	)

	if err := totalItemsForwardQuery.Error; err != nil {
		return pagination.PageInfo{}, fmt.Errorf("count items for forward pagination: %w", err)
	}

	if err := totalItemsBackwardsQuery.Error; err != nil {
		return pagination.PageInfo{}, fmt.Errorf("count items for backwards pagination: %w", err)
	}

	return pagination.PageInfo{
		HasPreviousPage: totalItemsBackwards > itemsCount,
		HasNextPage:     totalItemsForward > 0,
	}, nil
}

package pagination

import (
	"context"
	"fmt"
	"reflect"
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

	var totalCount int64
	if err := pageInfoSession.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}

	var (
		paginatedItems = make([]Item[T], len(items))
		startCursor    *Cursor
		endCursor      *Cursor
	)

	for i := range items {
		var (
			cursor Cursor
			err    error
		)
		// Need to pass address of item, otherwise it will not work for structs
		// containing array, as they will be marked unaddressable.
		v := reflect.ValueOf(items[i])
		if v.Kind() == reflect.Ptr {
			cursor, err = computeItemCursor(items[i])
		} else {
			// If obj is not a pointer, create a pointer to it
			ptr := reflect.New(v.Type())
			ptr.Elem().Set(v)
			cursor, err = computeItemCursor(ptr.Interface())
		}

		if err != nil {
			return nil, fmt.Errorf("compute cursor: %w", err)
		}

		if i == 0 {
			startCursor = &cursor
		}

		if i == len(items)-1 {
			endCursor = &cursor
		}

		paginatedItems[i] = Item[T]{
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

	pageInfo.TotalCount = int(totalCount)

	if len(paginatedItems) > 0 {
		pageInfo.StartCursor = ptr.To(paginatedItems[0].Cursor)
		pageInfo.EndCursor = ptr.To(paginatedItems[len(paginatedItems)-1].Cursor)
	}

	return &Page[T]{
		Items: paginatedItems,
		Info:  pageInfo,
	}, nil
}

// nolint: gocognit, gocyclo
func queryItems[T Tabler](ses *gorm.DB, pagination Arguments) ([]T, error) {
	var model T
	table := model.TableName()

	var items []T

	pagSession := ses.Session(&gorm.Session{})

	if pagination.First == nil && pagination.Last == nil {
		pagSession = pagSession.Order("created_at DESC")

		if pagination.afterCursor != nil {
			pagSession = pagSession.Where(
				fmt.Sprintf("%s.created_at < ?", table),
				pagination.afterCursor.CreatedAt,
			)
		}

		if err := pagSession.Find(&items).Error; err != nil {
			return nil, fmt.Errorf("find items: %w", err)
		}
	}

	// First/After
	if pagination.First != nil {
		pagSession = pagSession.Order(
			fmt.Sprintf("%s.created_at DESC", table),
		).Limit(*pagination.First)

		if pagination.afterCursor != nil {
			pagSession = pagSession.Where(
				fmt.Sprintf("%s.created_at < ?", table),
				pagination.afterCursor.CreatedAt,
			)
		}

		if err := pagSession.Find(&items).Error; err != nil {
			return nil, fmt.Errorf("find items: %w", err)
		}
	}

	// Last/Before
	if pagination.Last != nil {
		pagSession = pagSession.Order(
			fmt.Sprintf("%s.created_at ASC", table),
		).Limit(*pagination.Last)

		if pagination.beforeCursor != nil {
			pagSession = pagSession.Where(
				fmt.Sprintf("%s.created_at > ?", table),
				pagination.beforeCursor.CreatedAt,
			)
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

func hasNextPage[T Tabler](ses *gorm.DB, createdAt time.Time) (bool, error) {
	var model T
	condition := fmt.Sprintf("%s.created_at < ?", model.TableName())
	return hasMoreItems[T](ses, condition, createdAt)
}

func hasPreviousPage[T Tabler](ses *gorm.DB, createdAt time.Time) (bool, error) {
	var model T
	condition := fmt.Sprintf("%s.created_at > ?", model.TableName())
	return hasMoreItems[T](ses, condition, createdAt)
}

func hasMoreItems[T Tabler](ses *gorm.DB, condition string, createdAt time.Time) (bool, error) {
	var model T
	var result bool

	err := ses.Session(&gorm.Session{}).
		Model(model).
		Select("1").
		Where(condition, createdAt).
		Limit(1).
		Find(&result).Error
	if err != nil {
		return result, err
	}

	return result, nil
}

func getForwardPaginationPageInfo[T Tabler](
	db *gorm.DB,
	pagination Arguments,
	endCursor *Cursor,
) (PageInfo, error) {
	createdAt := time.Now()

	if endCursor != nil {
		createdAt = endCursor.CreatedAt
	} else if pagination.afterCursor != nil {
		// Case where zero items are fetched but with a cursor.
		createdAt = pagination.afterCursor.CreatedAt
	}

	var (
		pageInfo PageInfo
		err      error
	)

	pageInfo.HasNextPage, err = hasNextPage[T](db, createdAt)
	if err != nil {
		return PageInfo{}, fmt.Errorf("existence check for forward pagination: %w", err)
	}

	if pagination.afterCursor == nil {
		pageInfo.HasPreviousPage = false
		return pageInfo, nil
	}

	pageInfo.HasPreviousPage, err = hasPreviousPage[T](db, createdAt)
	if err != nil {
		return PageInfo{}, fmt.Errorf("existence check for forward pagination: %w", err)
	}

	return pageInfo, nil
}

func getBackwardPaginationPageInfo[T Tabler](
	db *gorm.DB,
	pagination Arguments,
	startCursor *Cursor,
) (PageInfo, error) {
	var createdAt time.Time

	if startCursor != nil {
		createdAt = startCursor.CreatedAt
	} else if pagination.beforeCursor != nil {
		// Case where zero items are fetched but with a cursor.
		createdAt = pagination.beforeCursor.CreatedAt
	}

	var (
		pageInfo PageInfo
		err      error
	)

	pageInfo.HasPreviousPage, err = hasPreviousPage[T](db, createdAt)
	if err != nil {
		return PageInfo{}, fmt.Errorf("existence check for backward pagination: %w", err)
	}

	if pagination.beforeCursor == nil {
		pageInfo.HasNextPage = false
		return pageInfo, nil
	}

	pageInfo.HasNextPage, err = hasNextPage[T](db, createdAt)
	if err != nil {
		return PageInfo{}, fmt.Errorf("existence check for backward pagination: %w", err)
	}

	return pageInfo, nil
}

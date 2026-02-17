package pagination_test

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/purposeinplay/go-commons/pagination"
	"github.com/purposeinplay/go-commons/psqldocker"
	"github.com/purposeinplay/go-commons/psqlutil"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"k8s.io/utils/ptr"
)

type user struct {
	ID   string  `gorm:"column:id;type:uuid;primaryKey"`
	Name *string `gorm:"column:name;type:text"`
	// nolint: revive
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
}

func (*user) TableName() string {
	return "users"
}

type machine struct {
	ID   uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Name *string   `gorm:"column:name;type:text"`
	// nolint: revive
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
}

func (machine) TableName() string {
	return "machines"
}

func userToCursor(u *user) *string {
	return ptr.To((&pagination.Cursor{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
	}).String())
}

func setupPsql(t *testing.T) *gorm.DB {
	req := require.New(t)

	const (
		psqlUser     = "postgres"
		psqlPassword = "postgres"
		psqlDB       = "postgres"
	)

	const schema = `
	CREATE TABLE users (
	    id 			UUID PRIMARY KEY,
	    name 		TEXT,
	    created_at 	TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	CREATE TABLE machines (
	    id 			UUID PRIMARY KEY,
	    name 		TEXT,
	    created_at 	TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);
	`

	psqlContainer := psqldocker.NewContainer(
		psqlUser,
		psqlPassword,
		psqlDB,
		psqldocker.WithExpiration(10),
		psqldocker.WithSQL(schema),
	)

	err := psqlContainer.Start()
	req.NoError(err)

	t.Cleanup(func() {
		err := psqlContainer.Close()
		req.NoError(err)
	})

	db, err := psqlutil.GormOpen(
		context.Background(),
		psqlutil.NewSlogLogger(slog.Default()),
		psqlutil.ConnectionConfig{
			Host:     "localhost",
			Port:     psqlContainer.Port(),
			User:     psqlUser,
			Password: psqlPassword,
			DBName:   psqlDB,
			SSLMode:  "disable",
		}.DSN(),
		false,
		nil,
	)
	req.NoError(err)

	db.Logger = db.Logger.LogMode(logger.Info)

	return db
}

func TestListPSQLPaginatedItems(t *testing.T) {
	t.Parallel()

	req := require.New(t)
	ctx := context.Background()

	db := setupPsql(t)

	users := make([]*user, 100)

	for i := 0; i < 100; i++ {
		id := uuid.UUID{}

		id[0] = byte(i)

		users[i] = &user{
			ID:        id.String(),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	err := db.Create(users).Error
	req.NoError(err)

	slices.Reverse(users)

	psqlPaginator := pagination.PSQLPaginator[*user]{
		DB: db,
	}

	tests := map[string]struct {
		params pagination.Arguments

		expectedError    require.ErrorAssertionFunc
		expectedUsers    []*user
		expectedPageInfo pagination.PageInfo
	}{
		"NoPagination": {
			params:        pagination.Arguments{},
			expectedUsers: users,
			expectedError: require.NoError,
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     false,
				StartCursor:     userToCursor(users[0]),
				EndCursor:       userToCursor(users[len(users)-1]),
				TotalCount:      100,
			},
		},
		"First3": {
			params: pagination.Arguments{
				First: ptr.To(3),
			},
			expectedError: require.NoError,
			expectedUsers: users[:3],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     userToCursor(users[0]),
				EndCursor:       userToCursor(users[2]),
				TotalCount:      100,
			},
		},
		"First3After3": {
			params: pagination.Arguments{
				First: ptr.To(3),
				After: userToCursor(users[2]),
			},
			expectedError: require.NoError,
			expectedUsers: users[3:6],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     true,
				StartCursor:     userToCursor(users[3]),
				EndCursor:       userToCursor(users[5]),
				TotalCount:      100,
			},
		},
		"First94After6": {
			params: pagination.Arguments{
				First: ptr.To(94),
				After: userToCursor(users[5]),
			},
			expectedError: require.NoError,
			expectedUsers: users[6:100],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     userToCursor(users[6]),
				EndCursor:       userToCursor(users[99]),
				TotalCount:      100,
			},
		},
		"Last3": {
			params: pagination.Arguments{
				Last: ptr.To(3),
			},
			expectedError: require.NoError,
			expectedUsers: users[97:100],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     userToCursor(users[97]),
				EndCursor:       userToCursor(users[99]),
				TotalCount:      100,
			},
		},
		"Last3Before6": {
			params: pagination.Arguments{
				Last:   ptr.To(3),
				Before: userToCursor(users[6]),
			},
			expectedError: require.NoError,
			expectedUsers: users[3:6],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     true,
				StartCursor:     userToCursor(users[3]),
				EndCursor:       userToCursor(users[5]),
				TotalCount:      100,
			},
		},
		"Last95Before95": {
			params: pagination.Arguments{
				Last:   ptr.To(95),
				Before: userToCursor(users[95]),
			},
			expectedError: require.NoError,
			expectedUsers: users[0:94],
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     userToCursor(users[0]),
				EndCursor:       userToCursor(users[94]),
				TotalCount:      100,
			},
		},
		"First0NoCursor": {
			params: pagination.Arguments{
				First: ptr.To(0),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
		"Last0NoCursor": {
			params: pagination.Arguments{
				Last: ptr.To(0),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
		"First0WithCursor": {
			params: pagination.Arguments{
				First: ptr.To(0),
				After: userToCursor(users[0]),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
		"Last0WithCursor": {
			params: pagination.Arguments{
				Last:   ptr.To(0),
				Before: userToCursor(users[99]),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
		"First0CursorAtEnd": {
			params: pagination.Arguments{
				First: ptr.To(0),
				After: userToCursor(users[99]),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
		"Last0CursorStart": {
			params: pagination.Arguments{
				Last:   ptr.To(0),
				Before: userToCursor(users[0]),
			},
			expectedError: require.NoError,
			expectedUsers: []*user{},
			expectedPageInfo: pagination.PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     nil,
				EndCursor:       nil,
				TotalCount:      100,
			},
		},
	}

	for name, test := range tests {
		test := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := require.New(t)

			page, err := psqlPaginator.ListItems(
				ctx,
				test.params,
			)
			test.expectedError(t, err)

			t.Logf("\nexpected users: %+v\nactual users: %+v", test.expectedUsers, page.Items)

			for i := range test.expectedUsers {
				req.Equal(
					test.expectedUsers[i],
					page.Items[i].Item,
					"id: %s, i: %d",
					page.Items[i].Item.ID,
					i,
				)
			}

			req.Equal(test.expectedPageInfo, page.Info)
		})
	}
}

func TestListPSQLPaginatedItemsWithWhereCondtion(t *testing.T) {
	req := require.New(t)
	ctx := context.Background()

	db := setupPsql(t)

	users := make([]*user, 6)

	first := "First"
	second := "Second"
	var name *string

	for i := 0; i < 6; i++ {
		id := uuid.UUID{}
		id[0] = byte(i)
		if i < 3 {
			name = &second
		} else {
			name = &first
		}
		users[i] = &user{
			ID:        id.String(),
			Name:      name,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	err := db.Create(users).Error
	req.NoError(err)

	slices.Reverse(users)

	params := pagination.Arguments{
		First: ptr.To(3),
	}
	db = db.Where("name = 'First'")
	psqlPaginator := pagination.PSQLPaginator[*user]{
		DB: db,
	}

	page, err := psqlPaginator.ListItems(
		ctx,
		params,
	)
	require.NoError(t, err)

	require.Len(t, page.Items, 3)
	for _, item := range page.Items {
		require.NotNil(t, item.Item.Name)
		require.Equal(t, "First", *item.Item.Name)
	}

	// First 3 users named "First"
	// Last 3 users named "Second"
	// Queried for first 3 users named "First"
	// Check that there are no next pages (users named "Second" not taken into account)
	require.Equal(t, page.Info, pagination.PageInfo{
		HasPreviousPage: false,
		HasNextPage:     false,
		StartCursor:     userToCursor(users[0]),
		EndCursor:       userToCursor(users[2]),
		TotalCount:      3,
	})
}

func TestListPSQLNonPointer(t *testing.T) {
	req := require.New(t)
	ctx := context.Background()

	db := setupPsql(t)

	name := "macbook"

	macbook := machine{
		Name:      &name,
		CreatedAt: time.Now().Add(time.Duration(1) * time.Second),
	}

	err := db.Create(&macbook).Error
	req.NoError(err)

	psqlPaginator := pagination.PSQLPaginator[machine]{
		DB: db,
	}

	page, err := psqlPaginator.ListItems(
		ctx,
		pagination.Arguments{
			First: ptr.To(1),
		},
	)
	require.NoError(t, err)

	require.Len(t, page.Items, 1)
	require.NotNil(t, page.Items[0].Item.Name)
	require.Equal(t, "macbook", *(page.Items[0].Item.Name))
}

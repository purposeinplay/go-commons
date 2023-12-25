package paginationpsql_test

import (
	"testing"
	"github.com/purposeinplay/go-commons/psqldocker"
	"github.com/stretchr/testify/require"
	"time"
	"github.com/purposeinplay/go-commons/psqlutil"
	"go.uber.org/zap"
	"context"
	"github.com/google/uuid"
	"github.com/purposeinplay/go-commons/pagination"
	"k8s.io/utils/ptr"
	"github.com/purposeinplay/go-commons/pagination/paginationpsql"
)

func TestListPSQLPaginatedItems(t *testing.T) {
	t.Parallel()

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

	type user struct {
		ID        string    `gorm:"column:id;type:uuid;primaryKey"`
		Name      *string   `gorm:"column:name;type:text"`
		CreatedAt time.Time `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
	}

	db, err := psqlutil.GormOpen(
		context.Background(),
		zap.NewExample(),
		"postgres",
		psqlutil.ComposeDSN(
			"localhost",
			psqlContainer.Port(),
			psqlUser,
			psqlPassword,
			psqlDB,
			"disable",
		),
		false,
	)
	req.NoError(err)

	users := make([]user, 100)

	for i := 0; i < 100; i++ {
		id := uuid.UUID{}

		id[0] = byte(i)

		users[i] = user{
			ID:        id.String(),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
	}

	err = db.Create(users).Error
	req.NoError(err)

	pagUsers, pageInfo, err := paginationpsql.ListPSQLPaginatedItems[user](
		db,
		pagination.Params{
			First: ptr.To(3),
		},
	)
	req.NoError(err)

	for i := range users[:3] {
		req.Equal(users[i], pagUsers[i].Item)
	}

	req.Equal(pagination.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
	}, pageInfo)

	pagUsers, pageInfo, err = paginationpsql.ListPSQLPaginatedItems[user](
		db,
		pagination.Params{
			First: ptr.To(3),
			After: &pagUsers[2].Cursor,
		},
	)
	req.NoError(err)

	for i := range users[3:6] {
		req.Equal(users[3:6][i], pagUsers[i].Item, "id: %s, i: %d", pagUsers[i].Item.ID, i)
	}

	req.Equal(pagination.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
	}, pageInfo)

	pagUsers, pageInfo, err = paginationpsql.ListPSQLPaginatedItems[user](
		db,
		pagination.Params{
			First: ptr.To(94),
			After: &pagUsers[2].Cursor,
		},
	)
	req.NoError(err)

	for i := range users[6:100] {
		req.Equal(users[6:100][i], pagUsers[i].Item, "id: %s, i: %d", pagUsers[i].Item.ID, i)
	}

	req.Equal(pagination.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
	}, pageInfo)
}

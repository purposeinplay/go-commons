package pagination

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

type entry struct {
	Member string
	Score  float64
}

func setupRedis(t *testing.T, members ...redis.Z) *redis.Client {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
	})

	if len(members) > 0 {
		ctx := context.Background()
		err := client.ZAdd(ctx, "testkey", members...).Err()
		require.NoError(t, err)
	}

	return client
}

func scoreCursor(score float64) *string {
	c := encodeScoreCursor(score)
	return &c
}

func TestListRedisPaginatedItems(t *testing.T) {
	t.Parallel()

	const total = 100
	const key = "testkey"

	members := make([]redis.Z, total)
	for i := 0; i < total; i++ {
		members[i] = redis.Z{
			Member: fmt.Sprintf("item-%03d", i),
			Score:  float64(i),
		}
	}

	// Items come back in descending score order.
	expectedAll := make([]entry, total)
	for i := 0; i < total; i++ {
		expectedAll[i] = entry{
			Member: fmt.Sprintf("item-%03d", total-1-i),
			Score:  float64(total - 1 - i),
		}
	}

	mapFn := func(member string, score float64) entry {
		return entry{Member: member, Score: score}
	}

	tests := map[string]struct {
		params           Arguments
		expectedItems    []entry
		expectedPageInfo PageInfo
	}{
		"NoPagination": {
			params:        Arguments{},
			expectedItems: expectedAll,
			expectedPageInfo: PageInfo{
				HasPreviousPage: false,
				HasNextPage:     false,
				StartCursor:     scoreCursor(99),
				EndCursor:       scoreCursor(0),
				TotalCount:      100,
			},
		},
		"First3": {
			params: Arguments{
				First: ptr.To(3),
			},
			expectedItems: expectedAll[:3],
			expectedPageInfo: PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     scoreCursor(99),
				EndCursor:       scoreCursor(97),
				TotalCount:      100,
			},
		},
		"First3AfterEndOfFirstPage": {
			params: Arguments{
				First: ptr.To(3),
				After: scoreCursor(97),
			},
			expectedItems: expectedAll[3:6],
			expectedPageInfo: PageInfo{
				HasPreviousPage: true,
				HasNextPage:     true,
				StartCursor:     scoreCursor(96),
				EndCursor:       scoreCursor(94),
				TotalCount:      100,
			},
		},
		"FirstRemainingAfterOffset6": {
			params: Arguments{
				First: ptr.To(94),
				After: scoreCursor(94),
			},
			expectedItems: expectedAll[6:100],
			expectedPageInfo: PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     scoreCursor(93),
				EndCursor:       scoreCursor(0),
				TotalCount:      100,
			},
		},
		"Last3": {
			params: Arguments{
				Last: ptr.To(3),
			},
			expectedItems: expectedAll[97:100],
			expectedPageInfo: PageInfo{
				HasPreviousPage: true,
				HasNextPage:     false,
				StartCursor:     scoreCursor(2),
				EndCursor:       scoreCursor(0),
				TotalCount:      100,
			},
		},
		"Last3BeforeScore93": {
			params: Arguments{
				Last:   ptr.To(3),
				Before: scoreCursor(93),
			},
			expectedItems: expectedAll[3:6],
			expectedPageInfo: PageInfo{
				HasPreviousPage: true,
				HasNextPage:     true,
				StartCursor:     scoreCursor(96),
				EndCursor:       scoreCursor(94),
				TotalCount:      100,
			},
		},
		"Last95BeforeScore4": {
			params: Arguments{
				Last:   ptr.To(95),
				Before: scoreCursor(4),
			},
			expectedItems: expectedAll[0:95],
			expectedPageInfo: PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     scoreCursor(99),
				EndCursor:       scoreCursor(5),
				TotalCount:      100,
			},
		},
		"LastExceedsBefore": {
			params: Arguments{
				Last:   ptr.To(10),
				Before: scoreCursor(96),
			},
			expectedItems: expectedAll[0:3],
			expectedPageInfo: PageInfo{
				HasPreviousPage: false,
				HasNextPage:     true,
				StartCursor:     scoreCursor(99),
				EndCursor:       scoreCursor(97),
				TotalCount:      100,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := require.New(t)

			client := setupRedis(t, members...)

			paginator := NewRedisSortedSetPaginator(
				client,
				key,
				mapFn,
			)

			page, err := paginator.ListItems(context.Background(), test.params)
			req.NoError(err)

			actual := make([]entry, len(page.Items))
			for i, item := range page.Items {
				actual[i] = item.Item
			}

			req.Equal(test.expectedItems, actual)
			req.Equal(test.expectedPageInfo, page.Info)
		})
	}
}

func TestListRedisPaginatedItems_EmptySet(t *testing.T) {
	t.Parallel()

	req := require.New(t)

	client := setupRedis(t)

	paginator := NewRedisSortedSetPaginator(
		client,
		"testkey",
		func(member string, score float64) entry {
			return entry{Member: member, Score: score}
		},
	)

	page, err := paginator.ListItems(context.Background(), Arguments{
		First: ptr.To(10),
	})
	req.NoError(err)
	req.Empty(page.Items)
	req.Equal(PageInfo{}, page.Info)
}

func TestListRedisPaginatedItems_InvalidCursor(t *testing.T) {
	t.Parallel()

	client := setupRedis(t, redis.Z{Member: "a", Score: 1})

	paginator := NewRedisSortedSetPaginator(
		client,
		"testkey",
		func(member string, score float64) entry {
			return entry{Member: member, Score: score}
		},
	)

	t.Run("InvalidAfterCursor", func(t *testing.T) {
		t.Parallel()

		_, err := paginator.ListItems(context.Background(), Arguments{
			First: ptr.To(1),
			After: ptr.To("not-valid-base64!@#$"),
		})
		require.Error(t, err)
	})

	t.Run("InvalidBeforeCursor", func(t *testing.T) {
		t.Parallel()

		_, err := paginator.ListItems(context.Background(), Arguments{
			Last:   ptr.To(1),
			Before: ptr.To("not-valid-base64!@#$"),
		})
		require.Error(t, err)
	})
}

func TestListRedisPaginatedItems_SequentialPagination(t *testing.T) {
	t.Parallel()

	req := require.New(t)

	members := make([]redis.Z, 10)
	for i := 0; i < 10; i++ {
		members[i] = redis.Z{
			Member: fmt.Sprintf("item-%d", i),
			Score:  float64(i),
		}
	}

	client := setupRedis(t, members...)

	mapFn := func(member string, score float64) entry {
		return entry{Member: member, Score: score}
	}

	paginator := NewRedisSortedSetPaginator(client, "testkey", mapFn)
	ctx := context.Background()

	page1, err := paginator.ListItems(ctx, Arguments{
		First: ptr.To(3),
	})
	req.NoError(err)
	req.Len(page1.Items, 3)
	req.Equal("item-9", page1.Items[0].Item.Member)
	req.Equal("item-7", page1.Items[2].Item.Member)
	req.True(page1.Info.HasNextPage)
	req.False(page1.Info.HasPreviousPage)

	page2, err := paginator.ListItems(ctx, Arguments{
		First: ptr.To(3),
		After: page1.Info.EndCursor,
	})
	req.NoError(err)
	req.Len(page2.Items, 3)
	req.Equal("item-6", page2.Items[0].Item.Member)
	req.Equal("item-4", page2.Items[2].Item.Member)
	req.True(page2.Info.HasNextPage)
	req.True(page2.Info.HasPreviousPage)

	page3, err := paginator.ListItems(ctx, Arguments{
		First: ptr.To(3),
		After: page2.Info.EndCursor,
	})
	req.NoError(err)
	req.Len(page3.Items, 3)
	req.Equal("item-3", page3.Items[0].Item.Member)
	req.Equal("item-1", page3.Items[2].Item.Member)
	req.True(page3.Info.HasNextPage)
	req.True(page3.Info.HasPreviousPage)

	page4, err := paginator.ListItems(ctx, Arguments{
		First: ptr.To(3),
		After: page3.Info.EndCursor,
	})
	req.NoError(err)
	req.Len(page4.Items, 1)
	req.Equal("item-0", page4.Items[0].Item.Member)
	req.False(page4.Info.HasNextPage)
	req.True(page4.Info.HasPreviousPage)

	backPage, err := paginator.ListItems(ctx, Arguments{
		Last:   ptr.To(3),
		Before: ptr.To(page3.Items[0].Cursor),
	})
	req.NoError(err)
	req.Len(backPage.Items, 3)
	req.Equal("item-6", backPage.Items[0].Item.Member)
	req.Equal("item-4", backPage.Items[2].Item.Member)
	req.True(backPage.Info.HasPreviousPage)
	req.True(backPage.Info.HasNextPage)
}

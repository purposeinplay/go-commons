package pagination

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var _ Paginator[any] = (*RedisSortedSetPaginator[any])(nil)

// RedisSortedSetPaginator implements Paginator for Redis sorted sets,
// returning items in descending score order with score-based keyset cursors.
type RedisSortedSetPaginator[T any] struct {
	client *redis.Client
	key    string
	mapFn  func(member string, score float64) T
}

// NewRedisSortedSetPaginator creates a paginator that reads from the given
// Redis sorted set key. mapFn converts each (member, score) pair into T.
func NewRedisSortedSetPaginator[T any](
	client *redis.Client,
	key string,
	mapFn func(member string, score float64) T,
) *RedisSortedSetPaginator[T] {
	return &RedisSortedSetPaginator[T]{
		client: client,
		key:    key,
		mapFn:  mapFn,
	}
}

// ListItems returns a paginated slice of items from the sorted set.
func (p *RedisSortedSetPaginator[T]) ListItems(
	ctx context.Context,
	args Arguments,
) (*Page[T], error) {
	total, err := p.client.ZCard(ctx, p.key).Result()
	if err != nil {
		return nil, fmt.Errorf("get total count: %w", err)
	}

	if total == 0 {
		return &Page[T]{}, nil
	}

	members, err := p.queryMembers(ctx, args, total)
	if err != nil {
		return nil, err
	}

	items := make([]T, len(members))
	scores := make([]float64, len(members))

	for i, m := range members {
		items[i] = p.mapFn(m.Member.(string), m.Score)
		scores[i] = m.Score
	}

	page, err := p.buildPage(ctx, items, scores, args, int(total))
	if err != nil {
		return nil, err
	}

	return page, nil
}

func (p *RedisSortedSetPaginator[T]) queryMembers(
	ctx context.Context,
	args Arguments,
	total int64,
) ([]redis.Z, error) {
	switch {
	case args.First != nil:
		return p.queryForward(ctx, args)

	case args.Last != nil:
		return p.queryBackward(ctx, args)

	default:
		members, err := p.client.ZRevRangeWithScores(
			ctx, p.key, 0, total-1,
		).Result()
		if err != nil {
			return nil, fmt.Errorf("get full range: %w", err)
		}

		return members, nil
	}
}

func (p *RedisSortedSetPaginator[T]) queryForward(
	ctx context.Context,
	args Arguments,
) ([]redis.Z, error) {
	max := "+inf"

	if args.After != nil {
		score, err := decodeScoreCursor(*args.After)
		if err != nil {
			return nil, fmt.Errorf("invalid after cursor: %w", err)
		}

		max = fmt.Sprintf("(%s", formatScore(score))
	}

	members, err := p.client.ZRevRangeByScoreWithScores(
		ctx, p.key, &redis.ZRangeBy{
			Min:   "-inf",
			Max:   max,
			Count: int64(*args.First),
		},
	).Result()
	if err != nil {
		return nil, fmt.Errorf("get forward range: %w", err)
	}

	return members, nil
}

func (p *RedisSortedSetPaginator[T]) queryBackward(
	ctx context.Context,
	args Arguments,
) ([]redis.Z, error) {
	min := "-inf"

	if args.Before != nil {
		score, err := decodeScoreCursor(*args.Before)
		if err != nil {
			return nil, fmt.Errorf("invalid before cursor: %w", err)
		}

		min = fmt.Sprintf("(%s", formatScore(score))
	}

	members, err := p.client.ZRangeByScoreWithScores(
		ctx, p.key, &redis.ZRangeBy{
			Min:   min,
			Max:   "+inf",
			Count: int64(*args.Last),
		},
	).Result()
	if err != nil {
		return nil, fmt.Errorf("get backward range: %w", err)
	}

	slices.Reverse(members)

	return members, nil
}

func (p *RedisSortedSetPaginator[T]) buildPage(
	ctx context.Context,
	items []T,
	scores []float64,
	args Arguments,
	total int,
) (*Page[T], error) {
	pageItems := make([]Item[T], len(items))
	for i, item := range items {
		pageItems[i] = Item[T]{
			Item:   item,
			Cursor: encodeScoreCursor(scores[i]),
		}
	}

	var startCursor, endCursor *string

	if len(pageItems) > 0 {
		startCursor = &pageItems[0].Cursor
		endCursor = &pageItems[len(pageItems)-1].Cursor
	}

	hasNext, err := p.hasNextPage(ctx, args, scores)
	if err != nil {
		return nil, err
	}

	hasPrev, err := p.hasPreviousPage(ctx, args, scores)
	if err != nil {
		return nil, err
	}

	return &Page[T]{
		Items: pageItems,
		Info: PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: hasPrev,
			StartCursor:     startCursor,
			EndCursor:       endCursor,
			TotalCount:      total,
		},
	}, nil
}

func (p *RedisSortedSetPaginator[T]) hasNextPage(
	ctx context.Context,
	args Arguments,
	scores []float64,
) (bool, error) {
	if len(scores) == 0 {
		if args.First != nil && args.After != nil {
			return false, nil
		}

		if args.Last != nil {
			return args.Before != nil, nil
		}

		return false, nil
	}

	lastScore := scores[len(scores)-1]

	count, err := p.client.ZCount(
		ctx, p.key, "-inf",
		fmt.Sprintf("(%s", formatScore(lastScore)),
	).Result()
	if err != nil {
		return false, fmt.Errorf("check next page: %w", err)
	}

	return count > 0, nil
}

func (p *RedisSortedSetPaginator[T]) hasPreviousPage(
	ctx context.Context,
	args Arguments,
	scores []float64,
) (bool, error) {
	if len(scores) == 0 {
		if args.Last != nil && args.Before != nil {
			return false, nil
		}

		if args.First != nil {
			return args.After != nil, nil
		}

		return false, nil
	}

	firstScore := scores[0]

	count, err := p.client.ZCount(
		ctx, p.key,
		fmt.Sprintf("(%s", formatScore(firstScore)),
		"+inf",
	).Result()
	if err != nil {
		return false, fmt.Errorf("check previous page: %w", err)
	}

	return count > 0, nil
}

func encodeScoreCursor(score float64) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(formatScore(score)),
	)
}

func decodeScoreCursor(cursor string) (float64, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("decode cursor base64: %w", err)
	}

	return strconv.ParseFloat(string(data), 64)
}

func formatScore(score float64) string {
	return strconv.FormatFloat(score, 'f', -1, 64)
}

package blockingqueue_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/blockingqueue"
)

func TestBlocking(t *testing.T) {
	t.Parallel()

	ids := []string{"0", "1", "2", "3"}
	ctx := context.Background()

	t.Run("Reset", func(t *testing.T) {
		i := is.New(t)

		blockingQueue := blockingqueue.New(ids)

		i.Equal("0", blockingQueue.Take(ctx))
		i.Equal("1", blockingQueue.Take(ctx))

		blockingQueue.Refill()

		i.Equal("2", blockingQueue.Take(ctx))

		blockingQueue.Refill()

		i.Equal("3", blockingQueue.Take(ctx))
		i.Equal("0", blockingQueue.Take(ctx))
		i.Equal("1", blockingQueue.Take(ctx))
		i.Equal("2", blockingQueue.Take(ctx))

		ctx, cancelCtx := context.WithTimeout(ctx, time.Millisecond)
		defer cancelCtx()

		i.Equal("", blockingQueue.Take(ctx))
	})

	t.Run("Consistency", func(t *testing.T) {
		i := is.New(t)

		const lenElements = 100

		ids := make([]int, lenElements)

		for i := 1; i <= lenElements; i++ {
			ids[i-1] = i
		}

		blockingQueue := blockingqueue.New(ids)

		result := make([]int, 0, lenElements)

		var (
			wg          sync.WaitGroup
			resultMutex sync.Mutex
		)

		wg.Add(lenElements)

		go blockingQueue.Refill()

		for i := 0; i < lenElements; i++ {
			go func() {
				elem := blockingQueue.Take(ctx)

				resultMutex.Lock()
				result = append(result, elem)
				resultMutex.Unlock()

				defer wg.Done()
			}()
		}

		wg.Wait()

		sort.SliceStable(result, func(i, j int) bool {
			return result[i] < result[j]
		})

		i.Equal(ids, result)
	})

	t.Run("SequentialIteration", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		blockingQueue := blockingqueue.New(ids)

		for j := range ids {
			id := blockingQueue.Take(ctx)

			i.Equal(ids[j], id)
		}
	})

	t.Run("CancelContext", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		blockingQueue := blockingqueue.New(ids)

		for range ids {
			blockingQueue.Take(ctx)
		}

		ctx, cancelCtx := context.WithTimeout(ctx, time.Millisecond)
		defer cancelCtx()

		e := blockingQueue.Take(ctx)

		i.Equal("", e)
	})

	t.Run("Refill", func(t *testing.T) {
		t.Parallel()

		t.Run("SequentialReset", func(t *testing.T) {
			t.Parallel()

			const noRoutines = 100

			for i := 1; i <= noRoutines; i++ {
				i := i

				t.Run(
					fmt.Sprintf("%dRoutinesWaiting", i),
					func(t *testing.T) {
						t.Parallel()

						testResetOnMultipleRoutinesFunc2[string](ctx, ids, i)(t)
					},
				)
			}
		})
	})
}

func testResetOnMultipleRoutinesFunc2[T any](
	ctx context.Context,
	ids []T,
	totalRoutines int,
) func(t *testing.T) {
	// nolint: thelper // not a test helper
	return func(t *testing.T) {
		blockingQueue := blockingqueue.New(ids)

		for range ids {
			blockingQueue.Take(ctx)
		}

		var wg sync.WaitGroup

		wg.Add(totalRoutines)

		retrievedID := make(chan T, len(ids))

		for routineIdx := 0; routineIdx < totalRoutines; routineIdx++ {
			go func(k int) {
				defer wg.Done()

				t.Logf("start routine %d", k)

				var id T

				defer func() {
					t.Logf("done routine %d, id %v", k, id)
				}()

				id = blockingQueue.Take(ctx)

				retrievedID <- id
			}(routineIdx)
		}

		time.Sleep(time.Millisecond)

		t.Log("reset")

		blockingQueue.Refill()

		counter := 0

		for range retrievedID {
			counter++

			t.Logf(
				"counter: %d, reset: %t",
				counter,
				counter%len(ids) == 0,
			)

			if counter == totalRoutines {
				break
			}

			if counter%len(ids) == 0 {
				blockingQueue.Refill()
			}
		}

		wg.Wait()
	}
}

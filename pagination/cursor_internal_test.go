package pagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestComputeCursor(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		item any

		expectedCursor string
		expectedError  require.ErrorAssertionFunc
	}{
		"NoID": {
			item: struct{ Foo string }{},
			expectedError: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrCursorFieldNotFound)
			},
			expectedCursor: "",
		},
		"NoCreatedAt": {
			item: struct {
				ID  string
				Foo string
			}{
				ID:  "3f6e8d5a-b972-4cb7-a741-ce03fe791439",
				Foo: "bar",
			},
			expectedError: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrCursorFieldNotFound)
			},
			expectedCursor: "",
		},
		"CreatedAtNotTime": {
			item: struct {
				ID        string
				CreatedAt string
			}{
				ID:        "3f6e8d5a-b972-4cb7-a741-ce03fe791439",
				CreatedAt: time.Now().String(),
			},
			expectedError: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrCursorInvalidValueType)
			},
			expectedCursor: "",
		},
		"CreatedAtTime": {
			item: struct {
				ID        string
				CreatedAt time.Time
			}{
				ID:        "3f6e8d5a-b972-4cb7-a741-ce03fe791439",
				CreatedAt: *timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			},
			expectedError: require.NoError,
			// nolint: revive
			expectedCursor: "M2Y2ZThkNWEtYjk3Mi00Y2I3LWE3NDEtY2UwM2ZlNzkxNDM5OjIwMjMtMTItMjBUMTM6NTY6MDNa",
		},
		"CreatedAtTimePtr": {
			item: struct {
				ID        string
				CreatedAt *time.Time
			}{
				ID:        "3f6e8d5a-b972-4cb7-a741-ce03fe791439",
				CreatedAt: timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			},
			expectedError: require.NoError,
			// nolint: revive
			expectedCursor: "M2Y2ZThkNWEtYjk3Mi00Y2I3LWE3NDEtY2UwM2ZlNzkxNDM5OjIwMjMtMTItMjBUMTM6NTY6MDNa",
		},
		"PtrCreatedAtTimePtr": {
			item: ptr.To(struct {
				ID        string
				CreatedAt *time.Time
			}{
				ID:        "3f6e8d5a-b972-4cb7-a741-ce03fe791439",
				CreatedAt: timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			}),
			expectedError: require.NoError,
			// nolint: revive
			expectedCursor: "M2Y2ZThkNWEtYjk3Mi00Y2I3LWE3NDEtY2UwM2ZlNzkxNDM5OjIwMjMtMTItMjBUMTM6NTY6MDNa",
		},
	}

	for name, test := range tests {
		test := test

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := require.New(t)

			cursor, err := computeItemCursor(test.item)

			test.expectedError(t, err)
			req.Equal(test.expectedCursor, cursor.String())
		})
	}
}

func timeMustParse(layout, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}

	return &t
}

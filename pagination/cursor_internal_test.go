package pagination

import (
	"reflect"
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
		"UUID": {
			item: ptr.To(struct {
				ID        []byte
				CreatedAt *time.Time
			}{
				ID:        []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
				CreatedAt: timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			}),
			expectedError: require.NoError,
			// nolint: revive
			expectedCursor: "MTIzNDU2NzgtOWFiYy1kZWYwLTEyMzQtNTY3ODlhYmNkZWYwOjIwMjMtMTItMjBUMTM6NTY6MDNa",
		},
		"SliceNotUUID": {
			item: ptr.To(struct {
				ID        []int
				CreatedAt *time.Time
			}{
				ID:        []int{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
				CreatedAt: timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			}),
			expectedError: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, ErrCursorInvalidValueType)
			},
			expectedCursor: "",
		},
		"CustomId": {
			item: ptr.To(StructWithCustomId{
				IdFieldName: "some_id",
				CreatedAt:   *timeMustParse(time.RFC3339, "2023-12-20T13:56:03Z"),
			}),
			expectedError:  require.NoError,
			expectedCursor: "c29tZV9pZDoyMDIzLTEyLTIwVDEzOjU2OjAzWg==",
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

type StructWithCustomId struct {
	IdFieldName string
	CreatedAt   time.Time
}

func (StructWithCustomId) TableName() string {
	return "struct_table"
}

func (StructWithCustomId) IdField() string {
	return "IdFieldName"
}

func timeMustParse(layout, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}

	return &t
}

func TestBytesToUUIDString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "valid UUID",
			input:    []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: "12345678-9abc-def0-1234-56789abcdef0",
		},
		{
			name:     "all zeros",
			input:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "00000000-0000-0000-0000-000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := bytesToUUIDString(tt.input)
			if actual != tt.expected {
				t.Errorf("bytesToUUIDString(%v) = %v, expected %v", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "valid UUID",
			input:    []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: true,
		},
		{
			name:     "too short",
			input:    []byte{0x12, 0x34, 0x56, 0x78},
			expected: false,
		},
		{
			name:     "not a slice",
			input:    "string",
			expected: false,
		},
		{
			name:     "not a byte slice",
			input:    []int32{1, 2},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isUUID(reflect.ValueOf(tt.input))
			if actual != tt.expected {
				t.Errorf("isUUID(%v) = %v, expected %v", tt.input, actual, tt.expected)
			}
		})
	}
}

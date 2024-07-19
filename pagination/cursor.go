package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const UUID_Length = 16

// Cursor defines the fields used to compose a cursor.
type Cursor struct {
	ID        string
	CreatedAt time.Time
}

// String returns the cursor fields encoded as a base64 string.
func (c *Cursor) String() string {
	if c.ID == "" || c.CreatedAt.IsZero() {
		return ""
	}

	cursorRaw := fmt.Sprintf("%s:%s", c.ID, c.CreatedAt.Format(time.RFC3339Nano))

	return base64.StdEncoding.EncodeToString([]byte(cursorRaw))
}

// ErrInvalidCursor is returned when the cursor is invalid.
var ErrInvalidCursor = errors.New("invalid Cursor")

// SetString decodes the cursor from a base64 string.
func (c *Cursor) SetString(text string) (*Cursor, error) {
	cursorRawBytes, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	id, createdAtRaw, ok := strings.Cut(string(cursorRawBytes), ":")
	if !ok {
		return nil, fmt.Errorf("%w: no separator provided", ErrInvalidCursor)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse created at: %w", err)
	}

	c.ID = id
	c.CreatedAt = createdAt

	return c, nil
}

func bytesToUUIDString(b []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16])
}

func isUUID(v reflect.Value) bool {
	if v.Kind() != reflect.Slice {
		return false
	}
	if v.Type().Elem().Kind() != reflect.Uint8 {
		return false
	}
	if len(v.Bytes()) != UUID_Length {
		return false
	}
	return true
}

// nolint: gocyclo
func computeItemCursor(obj any) (Cursor, error) {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	idFieldName := "ID"
	if tablerWithCustomId, ok := obj.(TablerWithId); ok {
		idFieldName = tablerWithCustomId.IdField()
	}

	idField := v.FieldByName(idFieldName)

	var (
		cursorID        string
		cursorCreatedAt time.Time
	)

	// nolint:exhaustive
	switch idField.Kind() {
	case reflect.String:
		cursorID = idField.String()
	case reflect.Slice:
		// Check if the slice is a UUID
		if isUUID(idField) {
			cursorID = bytesToUUIDString(idField.Bytes())
		} else {
			return Cursor{}, fmt.Errorf(
				"%w: ID: expected slice type to be []byte of length %d, actual: %s",
				ErrCursorInvalidValueType,
				UUID_Length,
				idField.Type(),
			)
		}
	case reflect.Invalid:
		return Cursor{}, fmt.Errorf("%w: ID", ErrCursorFieldNotFound)
	default:
		return Cursor{}, fmt.Errorf(
			"%w: ID: expected: %s, actual: %s",
			ErrCursorInvalidValueType,
			reflect.String,
			idField.Kind(),
		)
	}

	createdAtField := v.FieldByName("CreatedAt")

	// nolint:exhaustive
	switch createdAtField.Kind() {
	case timeKind:
		var ok bool

		cursorCreatedAt, ok = createdAtField.Interface().(time.Time)
		if !ok {
			return Cursor{}, fmt.Errorf(
				"%w: CreatedAt: expected: %s, actual: %s",
				ErrCursorInvalidValueType,
				"time.Time",
				reflect.TypeOf(createdAtField.Interface()).String(),
			)
		}

	case reflect.Ptr:
		createdAt, ok := createdAtField.Interface().(*time.Time)
		if !ok {
			return Cursor{}, fmt.Errorf(
				"%w: CreatedAt: expected: %s, actual: %s",
				ErrCursorInvalidValueType,
				"*time.Time",
				reflect.TypeOf(createdAtField.Interface()).String(),
			)
		}

		cursorCreatedAt = *createdAt

	case reflect.Invalid:
		return Cursor{}, fmt.Errorf("%w: CreatedAt", ErrCursorFieldNotFound)
	default:
		return Cursor{}, fmt.Errorf(
			"%w: CreatedAt: expected: %s, actual: %s",
			ErrCursorInvalidValueType,
			reflect.Ptr,
			idField.Kind(),
		)
	}

	return Cursor{
		ID:        cursorID,
		CreatedAt: cursorCreatedAt,
	}, nil
}

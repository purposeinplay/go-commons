package pagination

import (
	"encoding/base64"
	"time"
	"fmt"
	"strings"
	"errors"
	"reflect"
)

type Cursor struct {
	id        string
	createdAt time.Time
}

func (c *Cursor) String() string {
	cursorRaw := fmt.Sprintf("%s:%s", c.id, c.createdAt.Format(time.RFC3339Nano))

	return base64.StdEncoding.EncodeToString([]byte(cursorRaw))
}

var ErrInvalidCursor = errors.New("invalid Cursor")

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

	c.id = id
	c.createdAt = createdAt

	return c, nil
}

func computeStructCursor(obj any) (string, error) {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	idField := v.FieldByName("ID")

	var cursor Cursor

	switch idField.Kind() {
	case reflect.String:
		cursor.id = idField.String()
	case reflect.Invalid:
		return "", fmt.Errorf("%w: ID", ErrFieldNotFound)
	default:
		return "", fmt.Errorf(
			"%w: ID: expected: %s, actual: %s",
			ErrInvalidValueType,
			reflect.String,
			idField.Kind(),
		)
	}

	createdAtField := v.FieldByName("CreatedAt")

	switch createdAtField.Kind() {
	case timeKind:
		var ok bool

		cursor.createdAt, ok = createdAtField.Interface().(time.Time)
		if !ok {
			return "", fmt.Errorf(
				"%w: CreatedAt: expected: %s, actual: %s",
				ErrInvalidValueType,
				"time.Time",
				reflect.TypeOf(createdAtField.Interface()).String(),
			)
		}

	case reflect.Ptr:
		createdAt, ok := createdAtField.Interface().(*time.Time)
		if !ok {
			return "", fmt.Errorf(
				"%w: CreatedAt: expected: %s, actual: %s",
				ErrInvalidValueType,
				"*time.Time",
				reflect.TypeOf(createdAtField.Interface()).String(),
			)
		}

		cursor.createdAt = *createdAt

	case reflect.Invalid:
		return "", fmt.Errorf("%w: CreatedAt", ErrFieldNotFound)
	default:
		return "", fmt.Errorf(
			"%w: CreatedAt: expected: %s, actual: %s",
			ErrInvalidValueType,
			reflect.Ptr,
			idField.Kind(),
		)
	}

	return cursor.String(), nil
}

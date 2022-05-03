package value

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
)

var (
	// ensure Int implements valuer and scanner interface.
	_ sql.Scanner   = (*Int)(nil)
	_ driver.Valuer = (*Int)(nil)

	// ensure Int implements text marshaller and unmarshaler interface.
	_ encoding.TextMarshaler   = (*Int)(nil)
	_ encoding.TextUnmarshaler = (*Int)(nil)

	// ensure Int implements json marshaller and unmarshaler interface.
	_ json.Unmarshaler = (*Int)(nil)
	_ json.Marshaler   = (*Int)(nil)
)

// Int represents an integer
//
// ! This is intended to be used for storage and representation
// rather than for the big.Int behavior.
type Int struct {
	bigInt *big.Int
}

// NewIntFromString returns a new Int with the
// internal big.Int parsed from a string.
func NewIntFromString(s string) (*Int, error) {
	if s == "" {
		return nil, fmt.Errorf("%w: empty string value", ErrInvalidValue)
	}

	const base = 10

	b, ok := new(big.Int).SetString(s, base)
	if !ok {
		return nil, fmt.Errorf(
			"%w: string \"%s\" is not valid",
			ErrInvalidValue,
			s,
		)
	}

	return &Int{
		bigInt: b,
	}, nil
}

// NewIntFromInt64 returns a new Int with the
// internal big.Int set to v.
func NewIntFromInt64(v int64) *Int {
	return &Int{
		bigInt: big.NewInt(v),
	}
}

// NewIntFromBytes sets the internal bigInt type
// to the interpreted value of b.
func NewIntFromBytes(b []byte) (*Int, error) {
	if b == nil {
		return nil, fmt.Errorf("%w: nil bytes", ErrInvalidValue)
	}

	return &Int{
		bigInt: new(big.Int).SetBytes(b),
	}, nil
}

// NewIntFromBigInt sets the internal bigInt type to i.
func NewIntFromBigInt(i *big.Int) *Int {
	return &Int{
		bigInt: i,
	}
}

// IsValid returns true if the internal big.Int
// value is not nil.
func (v Int) IsValid() bool {
	return v.bigInt != nil
}

// IsGreaterThan compares v and x and returns:
// 		-1 if values can't be compared
// 		0 if v > x
// 		1 if v <= x
//
func (v Int) IsGreaterThan(x *Int) int {
	switch {
	case v.bigInt == nil || x.bigInt == nil:
		return -1

	case v.bigInt.Cmp(x.bigInt) == 1:
		return 0

	default:
		return 1
	}
}

// IsEqual compares v and x and returns:
// 		-1 if values can't be compared
// 		0 if v == x
// 		1 if v != x
//
func (v Int) IsEqual(x *Int) int {
	switch {
	case v.bigInt == nil || x.bigInt == nil:
		return -1

	case v.bigInt.Cmp(x.bigInt) == 0:
		return 0

	default:
		return 1
	}
}

// IsLesserThan compares v and x and returns:
// 		-1 if values can't be compared
// 		0 if v < x
// 		1 if v >= x
//
func (v Int) IsLesserThan(x *Int) int {
	switch {
	case v.bigInt == nil || x.bigInt == nil:
		return -1

	case v.bigInt.Cmp(x.bigInt) == -1:
		return 0

	default:
		return 1
	}
}

// SetBigInt is a wrapper over (*big.Int).Set..
// It sets the internal big.Int value to i.
func (v *Int) SetBigInt(i *big.Int) *Int {
	v.bigInt = new(big.Int).Set(i)

	return v
}

// SetBytes is a wrapper over (*big.Int).SetBytes.
//
// It interprets buf as the bytes of a big-endian unsigned
// integer, sets v.bigInt to that value, and returns v.
func (v *Int) SetBytes(buf []byte) *Int {
	v.bigInt = new(big.Int).SetBytes(buf)

	return v
}

// SetString is a wrapper over (*big.Int).SetString.
// It interprets the s and returns a boolean indicating
// the operation success.
func (v *Int) SetString(s string) (*Int, bool) {
	const base = 10

	bigInt, ok := new(big.Int).SetString(s, base)
	if !ok {
		return nil, false
	}

	v.bigInt = bigInt

	return v, ok
}

// BigInt returns the internal big.Int type.
func (v Int) BigInt() *big.Int {
	return v.bigInt
}

// Int64 it's a wrapper over (*big.Int).Int64.
//
// It returns the int64 representation of x.
// If x cannot be represented in an int64, the result is undefined.
func (v Int) Int64() int64 {
	if v.bigInt == nil {
		return 0
	}

	return v.bigInt.Int64()
}

// String returns the decimal representation of
// the internal big.Int.
func (v Int) String() string {
	return v.bigInt.String()
}

// Bytes returns the absolute value of the internal big.Int
// as a big-endian byte slice.
func (v Int) Bytes() []byte {
	return v.bigInt.Bytes()
}

// MarshalText implements the encoding.TextMarshaler interface.
func (v Int) MarshalText() ([]byte, error) {
	return v.bigInt.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (v *Int) UnmarshalText(text []byte) error {
	i := new(big.Int)

	err := i.UnmarshalText(text)
	if err != nil {
		return err
	}

	v.bigInt = i

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (v Int) MarshalJSON() ([]byte, error) {
	return v.bigInt.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Int) UnmarshalJSON(data []byte) error {
	i := new(big.Int)

	err := i.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	v.bigInt = i

	return nil
}

// Value defines how the Int is stored in the database.
func (v Int) Value() (driver.Value, error) {
	if v.IsValid() {
		return v.bigInt.String(), nil
	}

	return nil, nil
}

// Scan defines how the Int is read from the database.
func (v *Int) Scan(value interface{}) error {
	switch t := value.(type) {
	case int64:
		v.bigInt = new(big.Int).SetInt64(t)

	case []uint8:
		const base = 10

		var ok bool

		v.bigInt, ok = new(big.Int).SetString(string(t), base)
		if !ok {
			return fmt.Errorf(
				"%w: failed to load value to []uint8: %bigInt",
				ErrInvalidValue,
				value,
			)
		}

	case nil:
		v.bigInt = nil

	default:
		return fmt.Errorf(
			"%w: could not scan type %T into BigInt",
			ErrInvalidValue,
			t,
		)
	}

	return nil
}

// MustNewInt returns Int if err is nil and panics otherwise.
func MustNewInt(v *Int, err error) *Int {
	if err != nil {
		panic(err)
	}

	return v
}

package amount

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
)

var (
	// ensure ValueSubunit implements valuer and scanner interface.
	_ sql.Scanner   = (*ValueSubunit)(nil)
	_ driver.Valuer = (*ValueSubunit)(nil)

	// ensure ValueSubunit implements text marshaller and unmarshaler interface.
	_ encoding.TextMarshaler   = (*ValueSubunit)(nil)
	_ encoding.TextUnmarshaler = (*ValueSubunit)(nil)

	// ensure ValueSubunit implements json marshaller and unmarshaler interface.
	_ json.Unmarshaler = (*ValueSubunit)(nil)
	_ json.Marshaler   = (*ValueSubunit)(nil)
)

// ValueSubunit represents a value stored in its
// smallest denomination form.
// Example: wei for ethereum, satoshi fot btc, cents for dollars.
//
// ! This is intended to be used for storage and representation
// rather than for the big.Int behavior.
type ValueSubunit struct {
	bigInt *big.Int
}

// NewValueSubunitFromString returns a new ValueSubunit with the
// internal big.Int parsed from a string.
func NewValueSubunitFromString(s string) (*ValueSubunit, error) {
	const base = 10

	b, ok := new(big.Int).SetString(s, base)
	if !ok {
		return nil, fmt.Errorf(
			"%w: string \"%s\" is not valid",
			ErrInvalidValue,
			s,
		)
	}

	return &ValueSubunit{
		bigInt: b,
	}, nil
}

// NewValueSubunitFromInt64 returns a new ValueSubunit with the
// internal big.Int set to v.
func NewValueSubunitFromInt64(v int64) *ValueSubunit {
	return &ValueSubunit{
		bigInt: big.NewInt(v),
	}
}

// NewValueSubunitFromBytes sets the internal bigInt type
// to the interpreted value of b.
func NewValueSubunitFromBytes(b []byte) *ValueSubunit {
	return &ValueSubunit{
		bigInt: new(big.Int).SetBytes(b),
	}
}

// IsValid returns true if the internal big.Int
// value is not nil.
func (v ValueSubunit) IsValid() bool {
	return v.bigInt != nil
}

// IsGreaterThan compares v and x and returns:
// 		-1 if values can't be compared
// 		0 if v > x
// 		1 if v <= x
//
func (v ValueSubunit) IsGreaterThan(x *ValueSubunit) int {
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
func (v ValueSubunit) IsEqual(x *ValueSubunit) int {
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
func (v ValueSubunit) IsLesserThan(x *ValueSubunit) int {
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
func (v *ValueSubunit) SetBigInt(i *big.Int) *ValueSubunit {
	v.bigInt = new(big.Int).Set(i)

	return v
}

// SetBytes is a wrapper over (*big.Int).SetBytes.
//
// It interprets buf as the bytes of a big-endian unsigned
// integer, sets v.bigInt to that value, and returns v.
func (v *ValueSubunit) SetBytes(buf []byte) *ValueSubunit {
	v.bigInt = new(big.Int).SetBytes(buf)

	return v
}

// SetString is a wrapper over (*big.Int).SetString.
// It interprets the s and returns a boolean indicating
// the operation success.
func (v *ValueSubunit) SetString(s string) (*ValueSubunit, bool) {
	const base = 10

	bigInt, ok := new(big.Int).SetString(s, base)
	if !ok {
		return nil, false
	}

	v.bigInt = bigInt

	return v, ok
}

// BigInt returns the internal big.Int type.
func (v ValueSubunit) BigInt() *big.Int {
	return v.bigInt
}

// Int64 it's a wrapper over (*big.Int).Int64.
//
// It returns the int64 representation of x.
// If x cannot be represented in an int64, the result is undefined.
func (v ValueSubunit) Int64() int64 {
	if v.bigInt == nil {
		return 0
	}

	return v.bigInt.Int64()
}

// String returns the decimal representation of
// the internal big.Int.
func (v ValueSubunit) String() string {
	return v.bigInt.String()
}

// Bytes returns the absolute value of the internal big.Int
// as a big-endian byte slice.
func (v ValueSubunit) Bytes() []byte {
	return v.bigInt.Bytes()
}

// MarshalText implements the encoding.TextMarshaler interface.
func (v ValueSubunit) MarshalText() ([]byte, error) {
	return v.bigInt.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (v *ValueSubunit) UnmarshalText(text []byte) error {
	i := new(big.Int)

	err := i.UnmarshalText(text)
	if err != nil {
		return err
	}

	v.bigInt = i

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (v ValueSubunit) MarshalJSON() ([]byte, error) {
	return v.bigInt.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *ValueSubunit) UnmarshalJSON(data []byte) error {
	i := new(big.Int)

	err := i.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	v.bigInt = i

	return nil
}

// Value defines how the Int is stored in the database.
func (v ValueSubunit) Value() (driver.Value, error) {
	if v.IsValid() {
		return v.bigInt.String(), nil
	}

	return nil, nil
}

// Scan defines how the Int is read from the database.
func (v *ValueSubunit) Scan(value interface{}) error {
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

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
	// ensure NullInt implements valuer and scanner NullInterface.
	_ sql.Scanner   = (*NullInt)(nil)
	_ driver.Valuer = (*NullInt)(nil)

	// ensure NullInt implements text marshaller and unmarshaler NullInterface.
	_ encoding.TextMarshaler   = (*NullInt)(nil)
	_ encoding.TextUnmarshaler = (*NullInt)(nil)

	// ensure NullInt implements json marshaller and unmarshaler NullInterface.
	_ json.Unmarshaler = (*NullInt)(nil)
	_ json.Marshaler   = (*NullInt)(nil)
)

var (
	// NilNullInt has the `valid` property set to false.
	NilNullInt = NullInt{valid: false}

	// ZeroNullInt has the `valid` property set to true.
	ZeroNullInt = NullInt{valid: true}
)

// NullInt represents an NullInteger
//
// ! This is NullIntended to be used for storage and representation
// rather than for the big.NullInt behavior.
type NullInt struct {
	int Int

	valid bool
}

// NewNullIntFromString returns a new NullInt with the
// NullInternal big.NullInt parsed from a string.
func NewNullIntFromString(s string) (NullInt, error) {
	if s == "" {
		return NilNullInt, fmt.Errorf("%w: empty string value", ErrInvalidValue)
	}

	b, err := NewIntFromString(s)
	if err != nil {
		return NilNullInt, err
	}

	return NullInt{
		int:   b,
		valid: true,
	}, nil
}

// NewNullIntFromInt64 returns a new NullInt with the
// NullInternal big.NullInt set to v.
func NewNullIntFromInt64(v int64) NullInt {
	return NullInt{
		int:   NewIntFromInt64(v),
		valid: true,
	}
}

// NewNullIntFromUNullInt64 returns a new NullInt with the
// NullInternal big.NullInt set to v.
func NewNullIntFromUNullInt64(v uint64) NullInt {
	return NullInt{
		int:   NewIntFromUint64(v),
		valid: true,
	}
}

// NewNullIntFromBytes sets the NullInternal int type
// to the NullInterpreted value of b.
func NewNullIntFromBytes(b []byte) NullInt {
	return NullInt{
		int:   NewIntFromBytes(b),
		valid: true,
	}
}

// NewNullIntFromBigNullInt sets the NullInternal int type to i.
func NewNullIntFromBigNullInt(i *big.Int) NullInt {
	if i == nil {
		return NilNullInt
	}

	return NullInt{
		int:   NewIntFromBigInt(i),
		valid: true,
	}
}

// Neg sets v.int to -v.int.
func (v NullInt) Neg() NullInt {
	return NullInt{
		int:   v.int.Neg(),
		valid: true,
	}
}

// Add returns the sum between v and x.
func (v NullInt) Add(x NullInt) NullInt {
	return NullInt{
		int:   v.int.Add(x.int),
		valid: true,
	}
}

// Sub returns the difference between v and x.
func (v NullInt) Sub(x NullInt) NullInt {
	return NullInt{
		int:   v.int.Sub(x.int),
		valid: true,
	}
}

// Mul returns the product between v and x.
func (v NullInt) Mul(x NullInt) NullInt {
	return NullInt{
		int:   v.int.Mul(x.int),
		valid: true,
	}
}

// Div returns the division between v and x.
func (v NullInt) Div(x NullInt) NullInt {
	return NullInt{
		int:   v.int.Div(x.int),
		valid: true,
	}
}

// IsValid returns true if the NullInternal big.NullInt
// value is not nil.
func (v NullInt) IsValid() bool {
	return v.valid
}

// IsGreaterThan compares v and x and returns
// true if v is greater than x.
func (v NullInt) IsGreaterThan(x NullInt) bool {
	if v.valid && x.valid {
		return v.int.BigInt().Cmp(x.int.BigInt()) == 1
	}

	return false
}

// IsEqual compares v and x and returns
// true if v is equal to x.
func (v NullInt) IsEqual(x NullInt) bool {
	if v.valid && x.valid {
		return v.int.BigInt().Cmp(x.int.BigInt()) == 0
	}

	return v.valid == x.valid
}

// IsLesserThan compares v and x and returns
// true if v is lesser than x.
func (v NullInt) IsLesserThan(x NullInt) bool {
	if v.valid && x.valid {
		return v.int.BigInt().Cmp(x.int.BigInt()) == -1
	}

	return false
}

// SetBigInt is a wrapper over (*big.NullInt).Set..
// It sets the NullInternal big.NullInt value to i.
func (v *NullInt) SetBigInt(i *big.Int) NullInt {
	if i == nil {
		v.valid = false
		v.int = Int{}

		return *v
	}

	(&v.int).SetBigInt(i)
	v.valid = true

	return *v
}

// SetBytes is a wrapper over (*big.NullInt).SetBytes.
//
// It NullInterprets buf as the bytes of a big-endian unsigned
// NullInteger, sets v.int to that value, and returns v.
func (v *NullInt) SetBytes(buf []byte) NullInt {
	v.int = v.int.SetBytes(buf)
	v.valid = true

	return *v
}

// SetString is a wrapper over (*big.NullInt).SetString.
// It NullInterprets the s and returns a boolean indicating
// the operation success.
func (v *NullInt) SetString(s string) (NullInt, bool) {
	i, ok := (&v.int).SetString(s)

	v.valid = ok
	v.int = i

	return *v, ok
}

// BigInt returns the NullInternal big.NullInt type.
func (v NullInt) BigInt() *big.Int {
	return v.int.BigInt()
}

// Int64 it's a wrapper over (Int).Int64().
//
// It returns the Int64 representation of x.
// If x cannot be represented in an NullInt64, the result is undefined.
func (v NullInt) Int64() int64 {
	return v.int.Int64()
}

// String returns the decimal representation of
// the NullInternal big.NullInt.
func (v NullInt) String() string {
	return v.int.String()
}

// Bytes returns the absolute value of the NullInternal big.NullInt
// as a big-endian byte slice.
func (v NullInt) Bytes() []byte {
	return v.int.Bytes()
}

// MarshalText implements the encoding.TextMarshaler NullInterface.
func (v NullInt) MarshalText() ([]byte, error) {
	return v.int.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler NullInterface.
func (v *NullInt) UnmarshalText(text []byte) error {
	err := (&v.int).UnmarshalText(text)
	if err != nil {
		return err
	}

	v.valid = true

	return nil
}

// MarshalJSON implements the json.Marshaler NullInterface.
func (v NullInt) MarshalJSON() ([]byte, error) {
	return v.int.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler NullInterface.
func (v *NullInt) UnmarshalJSON(data []byte) error {
	err := (&v.int).UnmarshalJSON(data)
	if err != nil {
		return err
	}

	v.valid = true

	return nil
}

// Value defines how the NullInt is stored in the database.
func (v NullInt) Value() (driver.Value, error) {
	if v.IsValid() {
		return v.int.String(), nil
	}

	return nil, nil
}

// Scan defines how the NullInt is read from the database.
func (v *NullInt) Scan(value interface{}) error {
	switch t := value.(type) {
	case int64:
		v.int = NewIntFromInt64(t)
		v.valid = true

	case []uint8:
		i, err := NewIntFromString(string(t))
		if err != nil {
			return err
		}

		v.int = i
		v.valid = true

	case nil:
		v.int = ZeroInt
		v.valid = false

	default:
		return fmt.Errorf(
			"%w: could not scan type %T into NullInt",
			ErrInvalidValue,
			t,
		)
	}

	return nil
}

// MustNewNullInt returns NullInt if err is nil and panics otherwise.
func MustNewNullInt(v NullInt, err error) NullInt {
	if err != nil {
		panic(err)
	}

	return v
}

package amount

import (
	"fmt"
	"math/big"
)

// Amount represents a type storing information
// about a currency amount.
//
// ! The value is stored in its smallest denomination of the currency.
// Example: for dollars the amount is stored in cents:
// for 97.23 dollars, the value is 9723.
//
// Decimals represent the supported number of digits, after the decimals point.
// Example:
// - dollars decimals = 2 (smallest denomination: cents)
// - bitcoin decimals = 18 (smallest denomination: satoshi)
// - ethereum decimals = 18 (smallest denomination: wei).
type Amount struct {
	// value of the amount, stored as an int, in the smallest
	// denomination of the currency.
	value *big.Int

	// number of digits after the decimal point.
	decimals uint

	// shorthand for the currency.
	currencyCode string
}

// New creates a new amount from a *big.Int value.
// The value must be not nil.
func New(
	value *big.Int,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: nil value", ErrInvalidValue)
	}

	return &Amount{
		value:        value,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewFromStringValue creates a new amount from a string value.
// The value must be not empty.
// The value must be a valid int.
func NewFromStringValue(
	valueStr string,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	if valueStr == "" {
		return nil, fmt.Errorf("%w: empty string value", ErrInvalidValue)
	}

	const base = 10

	value, ok := new(big.Int).SetString(valueStr, base)
	if !ok {
		return nil, fmt.Errorf(
			"%w: string value \"%s\"",
			ErrInvalidValue,
			valueStr,
		)
	}

	return &Amount{
		value:        value,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewFromBytesValue creates a new amount from a []byte value.
// The value must be not nil.
func NewFromBytesValue(
	valueBytes []byte,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	if valueBytes == nil {
		return nil, fmt.Errorf("%w: nil bytes", ErrInvalidValue)
	}

	return &Amount{
		value:        new(big.Int).SetBytes(valueBytes),
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

func NewFromUnitStringAmount(
	unitValueStr string,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	if unitValueStr == "" {
		return nil, fmt.Errorf("%w: empty string value", ErrInvalidValue)
	}

	valueUnits, ok := new(big.Float).SetString(unitValueStr)
	if !ok {
		return nil, fmt.Errorf(
			"%w: string value \"%s\"",
			ErrInvalidValue,
			unitValueStr,
		)
	}

	value := fromUnits(valueUnits, decimals)

	return &Amount{
		value:        value,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// Must returns Amount if err is nil and panics otherwise.
func Must(amount *Amount, err error) *Amount {
	if err != nil {
		panic(err)
	}

	return amount
}

// Value returns the amount value in the *big.Int form.
func (a Amount) Value() *big.Int {
	return a.value
}

// Decimals returns the number of decimals for the amount.
func (a Amount) Decimals() uint {
	return a.decimals
}

// CurrencyCode returns the shorthand for the Currency Code of the Amount.
func (a Amount) CurrencyCode() string {
	return a.currencyCode
}

// AddBigInt sets the value of a.value to the sum a.value + v.
func (a *Amount) AddBigInt(v *big.Int) {
	a.value.Add(a.value, v)
}

// SubBigInt sets the value of a.value to the difference a.value - v.
func (a *Amount) SubBigInt(v *big.Int) {
	a.value.Sub(a.value, v)
}

// MulBigInt sets the value of a.value to the product a.value * v.
func (a *Amount) MulBigInt(v *big.Int) {
	a.value.Mul(a.value, v)
}

// DivBigInt sets the value of a.value to the quotient a.value / v.
// If v == 0, a division-by-zero run-time panic occurs.
func (a *Amount) DivBigInt(v *big.Int) {
	a.value.Div(a.value, v)
}

// ToUnits divides a.value / 10^decimals and returns a
// new big float containing the result.
func (a Amount) ToUnits() *big.Float {
	return toUnits(a.value, a.decimals)
}

func toUnits(value *big.Int, decimals uint) *big.Float {
	return new(big.Float).Quo(
		new(big.Float).SetInt(value),
		new(big.Float).SetInt(decimalsMultiplier(decimals)),
	)
}

func decimalsMultiplier(decimals uint) *big.Int {
	const ten = 10

	return new(big.Int).Exp(
		big.NewInt(ten),
		new(big.Int).SetUint64(uint64(decimals)),
		nil,
	)
}

func fromUnits(value *big.Float, decimals uint) *big.Int {
	i, _ := value.Mul(
		value,
		new(big.Float).SetInt(decimalsMultiplier(decimals)),
	).Int(nil)

	return i
}

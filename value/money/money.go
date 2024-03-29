package money

import (
	"fmt"
	"math/big"

	"github.com/purposeinplay/go-commons/value"
)

// Amount represents a type storing information
// about a monetary amount.
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
	value value.Int

	// number of digits after the decimal point.
	decimals uint

	// shorthand for the currency.
	currencyCode string
}

// GRPCMessageAmountString defines an interface that is implemented by
// GRPC messages carrying an amount.
type GRPCMessageAmountString interface {
	GetAmount() string
	GetDecimals() uint32
	GetCurrencyCode() string
}

// NewAmountFromValueInt creates a new money amount from a *Int value.
// The value must be not nil.
func NewAmountFromValueInt(
	v value.Int,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	return &Amount{
		value:        v,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewAmountFromStringValue creates a new amount from a string value.
// The value must be not empty.
// The value must be a valid int.
func NewAmountFromStringValue(
	valueStr string,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	v, err := value.NewIntFromString(valueStr)
	if err != nil {
		return nil, fmt.Errorf(
			"new value from string: %w",
			err,
		)
	}

	return NewAmountFromValueInt(v, decimals, currencyCode)
}

// NewAmountFromBytesValue creates a new amount from a []byte value.
// The value must be not nil.
func NewAmountFromBytesValue(
	valueBytes []byte,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	v := value.NewIntFromBytes(valueBytes)

	return NewAmountFromValueInt(v, decimals, currencyCode)
}

// NewAmountFromUnitStringAmount creates new Amount entity from a value that
// is in its largest denomination.
// The Amount value is calculated by multiplying unitValueStr * 10^decimals.
func NewAmountFromUnitStringAmount(
	unitValueStr string,
	decimals uint,
	currencyCode string,
) (*Amount, error) {
	if unitValueStr == "" {
		return nil, fmt.Errorf("%w: empty string value", value.ErrInvalidValue)
	}

	valueUnits, ok := new(big.Float).SetString(unitValueStr)
	if !ok {
		return nil, fmt.Errorf(
			"%w: string value \"%s\"",
			value.ErrInvalidValue,
			unitValueStr,
		)
	}

	v := value.NewIntFromBigInt(fromUnits(valueUnits, decimals))

	return NewAmountFromValueInt(v, decimals, currencyCode)
}

// NewAmountFromGRPCMessageAmountString creates a new Amount from
// an interface expected to be implemented by a GRPC message.
func NewAmountFromGRPCMessageAmountString(
	m GRPCMessageAmountString,
) (*Amount, error) {
	a, err := NewAmountFromStringValue(
		m.GetAmount(),
		uint(m.GetDecimals()),
		m.GetCurrencyCode(),
	)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// MustNewAmount returns Amount if err is nil and panics otherwise.
func MustNewAmount(amount *Amount, err error) *Amount {
	if err != nil {
		panic(err)
	}

	return amount
}

// Value returns the amount value in the *big.Int form.
func (a Amount) Value() value.Int {
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

// ToUnits divides a.value / 10^decimals and returns a
// new big float containing the result.
func (a Amount) ToUnits() *big.Float {
	return toUnits(a.value.BigInt(), a.decimals)
}

// ToUnitsString divides a.value / 10^decimals and returns a
// new string formatted to the given precision prec.
func (a Amount) ToUnitsString(prec uint8) string {
	return toUnits(a.value.BigInt(), a.decimals).Text('f', int(prec))
}

// toUnits returns value / 10^decimals.
func toUnits(v *big.Int, decimals uint) *big.Float {
	return new(big.Float).Quo(
		new(big.Float).SetInt(v),
		new(big.Float).SetInt(decimalsMultiplier(decimals)),
	)
}

// decimalsMultiplier returns 10^decimals.
func decimalsMultiplier(decimals uint) *big.Int {
	const ten = 10

	return new(big.Int).Exp(
		big.NewInt(ten),
		new(big.Int).SetUint64(uint64(decimals)),
		nil,
	)
}

// fromUnits returns value * 10^decimals.
func fromUnits(v *big.Float, decimals uint) *big.Int {
	i, _ := v.Mul(
		v,
		new(big.Float).SetInt(decimalsMultiplier(decimals)),
	).Int(nil)

	return i
}

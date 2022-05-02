package amount

import (
	"fmt"
	"math/big"
)

// Money represents a type storing information
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
type Money struct {
	// value of the amount, stored as an int, in the smallest
	// denomination of the currency.
	value *ValueSubunit

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

// NewMoney creates a new money amount from a *ValueSubunit value.
// The value must be not nil.
func NewMoney(
	value *ValueSubunit,
	decimals uint,
	currencyCode string,
) (*Money, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: nil value", ErrInvalidValue)
	}

	return &Money{
		value:        value,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewMoneyFromStringValue creates a new amount from a string value.
// The value must be not empty.
// The value must be a valid int.
func NewMoneyFromStringValue(
	valueStr string,
	decimals uint,
	currencyCode string,
) (*Money, error) {
	if valueStr == "" {
		return nil, fmt.Errorf("%w: empty string value", ErrInvalidValue)
	}

	value, err := NewValueSubunitFromString(valueStr)
	if err != nil {
		return nil, fmt.Errorf(
			"new value from string: %w",
			err,
		)
	}

	return &Money{
		value:        value,
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewMoneyFromBytesValue creates a new amount from a []byte value.
// The value must be not nil.
func NewMoneyFromBytesValue(
	valueBytes []byte,
	decimals uint,
	currencyCode string,
) (*Money, error) {
	if valueBytes == nil {
		return nil, fmt.Errorf("%w: nil bytes", ErrInvalidValue)
	}

	return &Money{
		value:        NewValueSubunitFromBytes(valueBytes),
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewMoneyFromUnitStringAmount creates new Money entity from a value that
// is in its largest denomination.
// The Money value is calculated by multiplying unitValueStr * 10^decimals.
func NewMoneyFromUnitStringAmount(
	unitValueStr string,
	decimals uint,
	currencyCode string,
) (*Money, error) {
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

	return &Money{
		value:        NewValueSubunitFromBigInt(value),
		decimals:     decimals,
		currencyCode: currencyCode,
	}, nil
}

// NewMoneyFromGRPCMessageAmountString creates a new Money from
// an interface expected to be implemented by a GRPC message.
func NewMoneyFromGRPCMessageAmountString(
	m GRPCMessageAmountString,
) (*Money, error) {
	a, err := NewMoneyFromStringValue(
		m.GetAmount(),
		uint(m.GetDecimals()),
		m.GetCurrencyCode(),
	)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// MustNewMoney returns Money if err is nil and panics otherwise.
func MustNewMoney(amount *Money, err error) *Money {
	if err != nil {
		panic(err)
	}

	return amount
}

// Value returns the amount value in the *big.Int form.
func (a Money) Value() *ValueSubunit {
	return a.value
}

// Decimals returns the number of decimals for the amount.
func (a Money) Decimals() uint {
	return a.decimals
}

// CurrencyCode returns the shorthand for the Currency Code of the Money.
func (a Money) CurrencyCode() string {
	return a.currencyCode
}

// ToUnits divides a.value / 10^decimals and returns a
// new big float containing the result.
func (a Money) ToUnits() *big.Float {
	return toUnits(a.value.bigInt, a.decimals)
}

// toUnits returns value / 10^decimals.
func toUnits(value *big.Int, decimals uint) *big.Float {
	return new(big.Float).Quo(
		new(big.Float).SetInt(value),
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
func fromUnits(value *big.Float, decimals uint) *big.Int {
	i, _ := value.Mul(
		value,
		new(big.Float).SetInt(decimalsMultiplier(decimals)),
	).Int(nil)

	return i
}

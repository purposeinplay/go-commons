package amount_test

import (
	"math/big"
	"testing"

	"github.com/matryer/is"
	"github.com/pkg/errors"
	"github.com/purposeinplay/go-commons/amount"
)

func TestConstructors(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.New(
				new(amount.ValueSubunit).SetBigInt(big.NewInt(100)),
				3,
				t.Name(),
			)
			i.NoErr(err)
		})

		t.Run("NilValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.New(
				nil,
				0,
				"",
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: nil value", err.Error())
		})
	})

	t.Run("NewFromStringValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewFromStringValue(
				"123456",
				3,
				t.Name(),
			)

			i.NoErr(err)
		})

		t.Run("InvalidStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewFromStringValue(
				"value",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: string value \"value\"", err.Error())
		})

		t.Run("EmptyStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewFromStringValue(
				"",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: empty string value", err.Error())
		})
	})

	t.Run("NewFromBytesValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			value := new(amount.ValueSubunit).SetBigInt(big.NewInt(1234))

			a, err := amount.NewFromBytesValue(
				value.Bytes(),
				3,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("value: %s", a.Value())

			i.True(a.Value().IsEqual(value))
		})

		t.Run("NilValueBytes", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewFromBytesValue(
				nil,
				0,
				"",
			)
			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: nil bytes", err.Error())
		})
	})

	t.Run("NewFromUnitStringAmount", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			a, err := amount.NewFromUnitStringAmount(
				"1234.56",
				5,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("value: %s", a.Value())

			i.True(
				a.Value().
					IsEqual(new(amount.ValueSubunit).
						SetBigInt(big.NewInt(123456000))),
			)
		})
	})

	t.Run("Must", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			defer func() {
				err := recover()
				i.True(err == nil)
			}()

			_ = amount.Must(amount.New(
				new(amount.ValueSubunit).SetBigInt(big.NewInt(10)),
				3,
				t.Name(),
			))
		})

		t.Run("InvalidValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			defer func() {
				err, ok := recover().(error)

				i.True(ok)

				i.True(errors.Is(err, amount.ErrInvalidValue))
				i.Equal("invalid value: nil value", err.Error())
			}()

			_ = amount.Must(amount.New(
				nil,
				0,
				"",
			))
		})
	})
}

func TestAmountMethods(t *testing.T) {
	t.Parallel()

	i := is.New(t)

	a, err := amount.New(
		new(amount.ValueSubunit).SetBigInt(big.NewInt(123456789)),
		3,
		t.Name())
	i.NoErr(err)

	t.Run("ToUnits", func(t *testing.T) {
		t.Parallel()

		i := i.New(t)

		i.Equal(
			"123456.789",
			a.ToUnits().String(),
		)
	})
}

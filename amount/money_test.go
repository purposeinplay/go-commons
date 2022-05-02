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

	t.Run("NewMoney", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoney(
				new(amount.ValueSubunit).SetBigInt(big.NewInt(100)),
				3,
				t.Name(),
			)
			i.NoErr(err)
		})

		t.Run("NilValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoney(
				nil,
				0,
				"",
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: nil value", err.Error())
		})
	})

	t.Run("NewMoneyFromStringValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoneyFromStringValue(
				"123456",
				3,
				t.Name(),
			)

			i.NoErr(err)
		})

		t.Run("InvalidStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoneyFromStringValue(
				"value",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("new value from string: "+
				"invalid value: string \"value\" is not valid",
				err.Error())
		})

		t.Run("EmptyStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoneyFromStringValue(
				"",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: empty string value", err.Error())
		})
	})

	t.Run("NewMoneyFromBytesValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			value := new(amount.ValueSubunit).SetBigInt(big.NewInt(1234))

			a, err := amount.NewMoneyFromBytesValue(
				value.Bytes(),
				3,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("value: %s", a.Value())

			i.True(a.Value().IsEqual(value) == 0)
		})

		t.Run("NilValueBytes", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := amount.NewMoneyFromBytesValue(
				nil,
				0,
				"",
			)
			i.True(errors.Is(err, amount.ErrInvalidValue))

			i.Equal("invalid value: nil bytes", err.Error())
		})
	})

	t.Run("NewMoneyFromUnitStringAmount", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			a, err := amount.NewMoneyFromUnitStringAmount(
				"1234.56",
				5,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("value: %s", a.Value())

			i.True(
				a.Value().
					IsEqual(new(amount.ValueSubunit).
						SetBigInt(big.NewInt(123456000))) == 0,
			)
		})
	})

	t.Run("MustNewMoney", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			defer func() {
				err := recover()
				i.True(err == nil)
			}()

			_ = amount.MustNewMoney(amount.NewMoney(
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

			_ = amount.MustNewMoney(amount.NewMoney(
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

	a, err := amount.NewMoney(
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

func TestComparisons(t *testing.T) {
	t.Parallel()

	var (
		one               = amount.NewValueSubunitFromInt64(1)
		two               = amount.NewValueSubunitFromInt64(2)
		nilInternalBigInt = new(amount.ValueSubunit)
	)

	t.Run("GreaterThan", func(t *testing.T) {
		t.Parallel()

		t.Run("Incomparable", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(-1, one.IsGreaterThan(nilInternalBigInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(0, two.IsGreaterThan(one))
		})

		t.Run("LesserOrEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(1, one.IsGreaterThan(two))
			i.Equal(1, one.IsGreaterThan(one))
		})
	})

	t.Run("LesserThan", func(t *testing.T) {
		t.Parallel()

		t.Run("Incomparable", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(-1, one.IsLesserThan(nilInternalBigInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(0, one.IsLesserThan(two))
		})

		t.Run("GreaterOrEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(1, two.IsLesserThan(one))
			i.Equal(1, two.IsLesserThan(two))
		})
	})

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		t.Run("Incomparable", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(-1, one.IsEqual(nilInternalBigInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(0, one.IsEqual(one))
		})

		t.Run("NotEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.Equal(1, two.IsEqual(one))
		})
	})
}

package money_test

import (
	"errors"
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/value"
	"github.com/purposeinplay/go-commons/value/money"
)

func TestConstructors(t *testing.T) {
	t.Parallel()

	t.Run("NewAmountFromValueInt", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromValueInt(
				value.NewIntFromInt64(100),
				3,
				t.Name(),
			)
			i.NoErr(err)
		})

		t.Run("ZeroValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromValueInt(
				value.ZeroInt,
				0,
				"",
			)
			i.NoErr(err)
		})
	})

	t.Run("NewAmountFromStringValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromStringValue(
				"123456",
				3,
				t.Name(),
			)

			i.NoErr(err)
		})

		t.Run("InvalidStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromStringValue(
				"value",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, value.ErrInvalidValue))

			i.Equal("new value from string: "+
				"invalid value: string \"value\" is not valid",
				err.Error())
		})

		t.Run("EmptyStringValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromStringValue(
				"",
				3,
				t.Name(),
			)

			i.True(errors.Is(err, value.ErrInvalidValue))

			i.Equal(
				"new value from string: invalid value: empty string value",
				err.Error(),
			)
		})
	})

	t.Run("NewAmountFromBytesValue", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(1234)

			a, err := money.NewAmountFromBytesValue(
				v.Bytes(),
				3,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("v: %s", a.Value())

			i.True(a.Value().IsEqual(v))
		})

		t.Run("NilValueBytes", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			_, err := money.NewAmountFromBytesValue(
				nil,
				0,
				"",
			)
			i.NoErr(err)
		})
	})

	t.Run("NewAmountFromUnitStringAmount", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			a, err := money.NewAmountFromUnitStringAmount(
				"1234.56",
				5,
				t.Name(),
			)
			i.NoErr(err)

			t.Logf("value: %s", a.Value())

			i.True(
				a.Value().
					IsEqual(value.NewIntFromInt64(123456000)),
			)
		})
	})

	t.Run("MustNewAmount", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			defer func() {
				err := recover()
				i.True(err == nil)
			}()

			_ = money.MustNewAmount(money.NewAmountFromValueInt(
				value.NewIntFromInt64(10),
				3,
				t.Name(),
			))
		})

		t.Run("ZeroValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			defer func() {
				err, ok := recover().(error)

				i.True(!ok)
				i.True(err == nil)
			}()

			_ = money.MustNewAmount(money.NewAmountFromValueInt(
				value.ZeroInt,
				0,
				"",
			))
		})
	})
}

func TestAmountMethods(t *testing.T) {
	t.Parallel()

	i := is.New(t)

	moneyAmount, err := money.NewAmountFromValueInt(
		value.NewIntFromInt64(123456789),
		3,
		t.Name())
	i.NoErr(err)

	t.Run("ToUnits", func(t *testing.T) {
		t.Parallel()

		i := i.New(t)

		i.Equal(
			"123456.789",
			moneyAmount.ToUnits().String(),
		)
	})

	t.Run("ToUnitsString", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		i.Equal(
			"123456.79",
			moneyAmount.ToUnitsString(2),
		)
	})
}

func TestComparisons(t *testing.T) {
	t.Parallel()

	var (
		one = value.NewIntFromInt64(1)
		two = value.NewIntFromInt64(2)
	)

	t.Run("GreaterThan", func(t *testing.T) {
		t.Parallel()

		t.Run("ZeroInt", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(one.IsGreaterThan(value.ZeroInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(two.IsGreaterThan(one))
		})

		t.Run("LesserOrEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(!one.IsGreaterThan(two))
			i.True(!one.IsGreaterThan(one))
		})
	})

	t.Run("LesserThan", func(t *testing.T) {
		t.Parallel()

		t.Run("Incomparable", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(!one.IsLesserThan(value.ZeroInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(one.IsLesserThan(two))
		})

		t.Run("GreaterOrEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(!two.IsLesserThan(one))
			i.True(!two.IsLesserThan(two))
		})
	})

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		t.Run("Incomparable", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(!one.IsEqual(value.ZeroInt))
		})

		t.Run("True", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(one.IsEqual(one))
		})

		t.Run("NotEqual", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			i.True(!two.IsEqual(one))
		})
	})
}

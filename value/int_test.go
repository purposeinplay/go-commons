package value_test

import (
	"strconv"
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/value"
)

func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("DefaultValue", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		var v value.Int

		vSQL, err := v.Value()
		i.NoErr(err)

		i.Equal("0", vSQL)
	})
}

func TestOperations(t *testing.T) {
	t.Parallel()

	t.Run("Neg", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		v := value.NewIntFromInt64(10)

		v = v.Neg()

		i.Equal(int64(-10), v.Int64())

		v = value.NewIntFromInt64(-10)

		i.Equal(int64(10), v.Neg().Int64())
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		i.Equal(
			int64(15),
			value.NewIntFromInt64(10).
				Add(value.NewIntFromInt64(5)).Int64(),
		)

		v := value.NewIntFromInt64(10)

		i.Equal(int64(20), (&v).Add(value.NewIntFromInt64(10)).Int64())
	})

	t.Run("Sub", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		i.Equal(
			int64(5),
			value.NewIntFromInt64(10).
				Sub(value.NewIntFromInt64(5)).Int64(),
		)
	})

	t.Run("Mul", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		i.Equal(
			int64(50),
			value.NewIntFromInt64(10).
				Mul(value.NewIntFromInt64(5)).Int64(),
		)
	})

	t.Run("Div", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		i.Equal(
			int64(2),
			value.NewIntFromInt64(10).
				Div(value.NewIntFromInt64(5)).Int64(),
		)
	})
}

func TestScan(t *testing.T) {
	t.Parallel()

	const (
		initialValueInt64 int64 = 100
		updatedValueInt64 int64 = 200
	)

	var (
		initialValueStr = strconv.FormatInt(initialValueInt64, 10)
		updatedValueStr = strconv.FormatInt(updatedValueInt64, 10)
	)

	t.Run("NilScannedValue", func(t *testing.T) {
		t.Parallel()

		t.Run("NotNilValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(initialValueInt64)
			i.True(v.IsEqual(
				value.MustNewInt(value.NewIntFromString("100"))))

			i.Equal(initialValueStr, v.String())

			err := v.Scan(nil)
			i.NoErr(err)

			i.Equal("0", v.String())
		})

		t.Run("DefaultValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			var v value.Int

			i.True(v.IsEqual(value.ZeroInt))

			err := v.Scan(nil)
			i.NoErr(err)

			i.True(v.IsEqual(value.ZeroInt))
		})
	})

	t.Run("ValidScannedValue", func(t *testing.T) {
		t.Parallel()

		t.Run("NotNilBytesValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(initialValueInt64)

			i.True(v.IsEqual(
				value.MustNewInt(value.NewIntFromString("100"))))

			err := v.Scan([]byte(updatedValueStr))
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NotZeroInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(initialValueInt64)

			i.True(v.IsEqual(
				value.MustNewInt(value.NewIntFromString("100"))))

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("ZeroInternalBigInt_NotZeroInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(value.Int)
			i.True(v != nil)

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NilValue_NotZeroInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			var v *value.Int

			defer func() {
				p := recover()

				// panic
				i.True(p != nil)
			}()

			_ = v.Scan(updatedValueInt64)
		})
	})
}

func TestEncodingText(t *testing.T) {
	t.Parallel()

	const (
		initialValueInt64 int64 = 100
		updatedValueInt64 int64 = 200
	)

	var (
		initialValueStr = strconv.FormatInt(initialValueInt64, 10)
		updatedValueStr = strconv.FormatInt(updatedValueInt64, 10)
	)

	t.Run("Text", func(t *testing.T) {
		t.Parallel()

		t.Run("Marshal", func(t *testing.T) {
			t.Parallel()

			t.Run("DefaultValue", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				var v value.Int

				txt, err := v.MarshalText()
				i.NoErr(err)

				i.Equal("0", string(txt))

				json, err := v.MarshalJSON()
				i.NoErr(err)

				i.Equal("0", string(json))
			})

			t.Run("ValidInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := value.NewIntFromInt64(initialValueInt64)

				txt, err := v.MarshalText()
				i.NoErr(err)

				i.Equal(initialValueStr, string(txt))

				json, err := v.MarshalJSON()
				i.NoErr(err)

				i.Equal(initialValueStr, string(json))
			})
		})

		t.Run("Unmarshal", func(t *testing.T) {
			t.Parallel()

			t.Run("ZeroInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := new(value.Int)

				err := v.UnmarshalText([]byte(initialValueStr))
				i.NoErr(err)

				i.Equal(initialValueStr, v.String())

				v = new(value.Int)

				err = v.UnmarshalJSON([]byte(initialValueStr))
				i.NoErr(err)

				i.Equal(initialValueStr, v.String())
			})

			t.Run("ValidInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := value.NewIntFromInt64(initialValueInt64)

				err := v.UnmarshalText([]byte(updatedValueStr))
				i.NoErr(err)

				i.Equal(updatedValueStr, v.String())

				v = value.NewIntFromInt64(initialValueInt64)

				err = v.UnmarshalJSON([]byte(updatedValueStr))
				i.NoErr(err)

				i.Equal(updatedValueStr, v.String())
			})
		})
	})
}

func TestZeroInt(t *testing.T) {
	t.Parallel()

	i := is.New(t)

	i.True(value.NewIntFromInt64(0).IsEqual(value.ZeroInt))
}

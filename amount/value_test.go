package amount_test

import (
	"strconv"
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/amount"
)

func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("NilInternalBigInt", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		v := new(amount.ValueSubunit)

		vSql, err := v.Value()
		i.NoErr(err)

		i.True(vSql == nil)
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

			v := amount.NewValueSubunitFromInt64(initialValueInt64)
			i.True(v != nil)

			i.Equal(initialValueStr, v.String())

			err := v.Scan(nil)
			i.NoErr(err)

			i.Equal("<nil>", v.String())
		})

		t.Run("NilValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			var v *amount.ValueSubunit

			i.True(v == nil)

			defer func() {
				p := recover()

				// panic
				i.True(p != nil)
			}()

			_ = v.Scan(nil)
		})

		t.Run("NilInternalBigInt", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(amount.ValueSubunit)

			defer func() {
				p := recover()

				// no panic
				i.True(p == nil)
			}()

			err := v.Scan(nil)
			i.NoErr(err)

			i.Equal("<nil>", v.String())
		})
	})

	t.Run("ValidScannedValue", func(t *testing.T) {
		t.Parallel()

		t.Run("NotNilBytesValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := amount.NewValueSubunitFromInt64(initialValueInt64)
			i.True(v != nil)

			err := v.Scan([]byte(updatedValueStr))
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := amount.NewValueSubunitFromInt64(initialValueInt64)
			i.True(v != nil)

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NilInternalBigInt_NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(amount.ValueSubunit)
			i.True(v != nil)

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NilValue_NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			var v *amount.ValueSubunit

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

			t.Run("NilInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := new(amount.ValueSubunit)

				txt, err := v.MarshalText()
				i.NoErr(err)

				i.Equal("<nil>", string(txt))

				json, err := v.MarshalJSON()
				i.NoErr(err)

				i.Equal("<nil>", string(json))
			})

			t.Run("ValidInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := amount.NewValueSubunitFromInt64(initialValueInt64)

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

			t.Run("NilInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := new(amount.ValueSubunit)

				err := v.UnmarshalText([]byte(initialValueStr))
				i.NoErr(err)

				i.Equal(initialValueStr, v.String())

				v = new(amount.ValueSubunit)

				err = v.UnmarshalJSON([]byte(initialValueStr))
				i.NoErr(err)

				i.Equal(initialValueStr, v.String())
			})

			t.Run("ValidInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := amount.NewValueSubunitFromInt64(initialValueInt64)

				err := v.UnmarshalText([]byte(updatedValueStr))
				i.NoErr(err)

				i.Equal(updatedValueStr, v.String())

				v = amount.NewValueSubunitFromInt64(initialValueInt64)

				err = v.UnmarshalJSON([]byte(updatedValueStr))
				i.NoErr(err)

				i.Equal(updatedValueStr, v.String())
			})
		})
	})
}

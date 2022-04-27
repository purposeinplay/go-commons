package amount_test

import (
	"math/big"
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/amount"
)

func TestCmp(t *testing.T) {
	t.Parallel()

	t.Run("NilValues", func(t *testing.T) {
	})
}

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

	t.Run("NilScannedValue", func(t *testing.T) {
		t.Parallel()

		t.Run("NotNilValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			const value int64 = 100

			v := new(amount.ValueSubunit).SetBigInt(big.NewInt(value))
			i.True(v != nil)

			i.Equal("100", v.String())

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

		const (
			initialValue      int64  = 100
			updatedValueStr   string = "200"
			updatedValueInt64 int64  = 200
		)

		t.Run("NotNilBytesValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(amount.ValueSubunit).SetBigInt(big.NewInt(initialValue))
			i.True(v != nil)

			err := v.Scan([]byte(updatedValueStr))
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(amount.ValueSubunit).SetBigInt(big.NewInt(initialValue))
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

				v := new(amount.ValueSubunit).SetBigInt(big.NewInt(100))

				txt, err := v.MarshalText()
				i.NoErr(err)

				i.Equal("100", string(txt))

				json, err := v.MarshalJSON()
				i.NoErr(err)

				i.Equal("100", string(json))
			})
		})

		t.Run("Unmarshal", func(t *testing.T) {
			t.Parallel()

			t.Run("NilInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := new(amount.ValueSubunit)

				err := v.UnmarshalText([]byte("100"))
				i.NoErr(err)

				i.Equal("100", v.String())

				v = new(amount.ValueSubunit)

				err = v.UnmarshalJSON([]byte("100"))
				i.NoErr(err)

				i.Equal("100", v.String())
			})

			t.Run("ValidInternalBigInteger", func(t *testing.T) {
				t.Parallel()

				i := is.New(t)

				v := new(amount.ValueSubunit).SetBigInt(big.NewInt(100))

				err := v.UnmarshalText([]byte("200"))
				i.NoErr(err)

				i.Equal("200", v.String())

				v = new(amount.ValueSubunit).SetBigInt(big.NewInt(100))

				err = v.UnmarshalJSON([]byte("200"))
				i.NoErr(err)

				i.Equal("200", v.String())
			})
		})
	})
}

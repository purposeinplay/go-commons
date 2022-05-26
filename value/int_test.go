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

		i.True(vSQL == nil)
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
			i.True(v.IsValid())

			i.Equal(initialValueStr, v.String())

			err := v.Scan(nil)
			i.NoErr(err)

			i.Equal("0", v.String())
		})

		t.Run("DefaultValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			var v value.Int

			i.True(!v.IsValid())

			err := v.Scan(nil)
			i.NoErr(err)

			i.True(!v.IsValid())
		})
	})

	t.Run("ValidScannedValue", func(t *testing.T) {
		t.Parallel()

		t.Run("NotNilBytesValue", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(initialValueInt64)
			i.True(v.IsValid())

			err := v.Scan([]byte(updatedValueStr))
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := value.NewIntFromInt64(initialValueInt64)
			i.True(v.IsValid())

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NilInternalBigInt_NotNilInt64Value", func(t *testing.T) {
			t.Parallel()

			i := is.New(t)

			v := new(value.Int)
			i.True(v != nil)

			err := v.Scan(updatedValueInt64)
			i.NoErr(err)

			i.Equal(updatedValueStr, v.String())
		})

		t.Run("NilValue_NotNilInt64Value", func(t *testing.T) {
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

			t.Run("NilInternalBigInteger", func(t *testing.T) {
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

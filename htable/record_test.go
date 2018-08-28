package htable

import (
	"reflect"
	"testing"

	"github.com/zeebo/gofaster/internal/assert"
	"github.com/zeebo/gofaster/pin"
)

func TestRecord(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		rec := newRecord([]byte("key"), []byte("value"))
		assert.Equal(t, rec.key, 3)
		assert.Equal(t, rec.val, 5)
		assert.Equal(t, string(rec.Key()), "key")
		assert.Equal(t, string(rec.Val()), "value")
	})

	t.Run("Only Basic", func(t *testing.T) {
		locationType := reflect.TypeOf(pin.Location{})
		rv := reflect.TypeOf(record{})

		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)

			// pin.Location is known to be fine
			if field.Type == locationType {
				continue
			}

			switch field.Type.Kind() {
			case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice,
				reflect.Struct, reflect.Func, reflect.Interface:

				t.Fatalf("field %q is non-basic", field.Name)
			}
		}
	})
}

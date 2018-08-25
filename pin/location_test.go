package pin

import (
	"testing"

	"github.com/zeebo/gofaster/internal/assert"
)

func TestLocation(t *testing.T) {
	t.Run("Extra", func(t *testing.T) {
		loc := newLocation(1, 2)

		assert.Equal(t, loc.Extra(), 0)
		assert.Equal(t, loc.id(), 1)
		assert.Equal(t, loc.index(), 2)

		loc2 := loc.WithExtra(1063)

		assert.Equal(t, loc.Extra(), 0)
		assert.Equal(t, loc.id(), 1)
		assert.Equal(t, loc.index(), 2)

		assert.Equal(t, loc2.Extra(), 1063)
		assert.Equal(t, loc2.id(), 1)
		assert.Equal(t, loc2.index(), 2)
	})
}

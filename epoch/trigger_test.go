package epoch

import (
	"testing"

	"github.com/zeebo/gofaster/internal/assert"
)

func TestTrigger(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		tr := newTrigger()

		ran := false
		assert.That(t, tr.Free())
		assert.That(t, tr.Store(8, func() { ran = true }))
		assert.Equal(t, tr.Epoch(), 8)

		assert.That(t, !tr.Run(7))
		assert.That(t, !ran)
		assert.That(t, !tr.Free())

		assert.That(t, tr.Run(8))
		assert.That(t, ran)
		assert.That(t, tr.Free())
	})

	t.Run("Swap", func(t *testing.T) {
		tr := newTrigger()

		ran1 := false
		assert.That(t, tr.Store(8, func() { ran1 = true }))
		assert.Equal(t, tr.Epoch(), 8)

		ran2 := false
		assert.That(t, tr.Swap(8, 9, func() { ran2 = true }))
		assert.That(t, ran1)
		assert.Equal(t, tr.Epoch(), 9)

		assert.That(t, tr.Run(9))
		assert.That(t, ran2)
	})
}

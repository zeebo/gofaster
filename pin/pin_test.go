package pin

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/assert"
)

func TestPin(t *testing.T) {
	h := epoch.AcquireHandle()
	defer epoch.ReleaseHandle(h)

	// make a pointer and attach a finalizer
	x := new([1024]byte)
	finalized := false
	runtime.SetFinalizer(x, func(*[1024]byte) { finalized = true })

	// pin the item, remove our reference, and GC
	loc := Pin(h, unsafe.Pointer(x))
	x = nil
	runtime.GC()

	// it shouldn't be finalized yet
	assert.That(t, !finalized)

	// unpin the item and GC twice
	Unpin(h, loc)
	runtime.GC()
	runtime.GC()

	// it should be finalized
	assert.That(t, finalized)
}

func BenchmarkPin(b *testing.B) {
	mem := unsafe.Pointer(new([1024]byte))

	b.Run("Pin+Unpin Same Handle", func(b *testing.B) {
		b.ReportAllocs()

		h := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h)

		for i := 0; i < b.N; i++ {
			loc := Pin(h, mem)
			Unpin(h, loc)
		}
	})

	b.Run("Pin+Unpin Different Handle", func(b *testing.B) {
		b.ReportAllocs()

		h1 := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h1)
		h2 := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h2)

		for i := 0; i < b.N; i++ {
			loc1, loc2, loc3 := Pin(h1, mem), Pin(h1, mem), Pin(h1, mem)
			Unpin(h2, loc1)
			Unpin(h2, loc2)
			Unpin(h2, loc3)
		}
	})

	b.Run("Pin+Unpin Same Handle Parallel", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			h := epoch.AcquireHandle()
			defer epoch.ReleaseHandle(h)

			for pb.Next() {
				loc := Pin(h, mem)
				Unpin(h, loc)
			}
		})
	})
}

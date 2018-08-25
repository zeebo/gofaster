package pin

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/assert"
	"github.com/zeebo/gofaster/internal/machine"
	"github.com/zeebo/gofaster/internal/pcg"
)

func TestPin(t *testing.T) {
	h := epoch.AcquireHandle()
	defer epoch.ReleaseHandle(h)

	// make a pointer and attach a finalizer
	x := new([1024]byte)
	finalized := uint64(0)
	runtime.SetFinalizer(x, func(*[1024]byte) {
		atomic.StoreUint64(&finalized, 1)
	})

	// pin the item, remove our reference, and GC
	loc := Pin(h, unsafe.Pointer(x))
	assert.Equal(t, Read(loc), unsafe.Pointer(x))
	x = nil
	runtime.GC()

	// it shouldn't be finalized yet
	assert.That(t, atomic.LoadUint64(&finalized) == 0)

	// unpin the item and GC twice
	Unpin(h, loc)
	runtime.GC()
	runtime.GC()

	// it should be finalized
	assert.That(t, atomic.LoadUint64(&finalized) == 1)
}

func BenchmarkPin(b *testing.B) {
	mem := unsafe.Pointer(new([1024]byte))

	b.Run("Same Handle", func(b *testing.B) {
		b.ReportAllocs()

		h := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h)

		for i := 0; i < b.N; i++ {
			loc := Pin(h, mem)
			Unpin(h, loc)
		}
	})

	b.Run("Different Handle", func(b *testing.B) {
		b.ReportAllocs()

		h1 := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h1)
		h2 := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h2)

		for i := 0; i < b.N; i++ {
			loc := Pin(h1, mem)
			Unpin(h2, loc)
		}
	})

	b.Run("Same Handle Parallel", func(b *testing.B) {
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

	b.Run("Different Handle Parallel", func(b *testing.B) {
		index := uint64(0)
		hs := make([]epoch.Handle, machine.MaxThreads)
		for i := range hs {
			hs[i] = epoch.AcquireHandle()
			defer epoch.ReleaseHandle(hs[i])
		}

		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			// panic if too many handles outstanding because Pin
			// cannot be called concurrently with the same handle.
			i := atomic.AddUint64(&index, 1) - 1
			h := hs[i]
			p := pcg.New(i, uint64(time.Now().UnixNano()))

			// start loopin!
			for pb.Next() {
				loc := Pin(h, mem)

				// Unpin can be called concurrently with the same handle
				// so we are safe to pick a random one to stress test.
				hu := hs[p.Uint32()%machine.MaxThreads]
				Unpin(hu, loc)
			}
		})
	})
}

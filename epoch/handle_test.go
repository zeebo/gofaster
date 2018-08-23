package epoch

import (
	"testing"
)

func BenchmarkHandle(b *testing.B) {
	b.ReportAllocs()

	b.Run("Acquire+Release", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h := AcquireHandle()
			ReleaseHandle(h)
		}
	})

	b.Run("Acquire+Release Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				h := AcquireHandle()
				ReleaseHandle(h)
			}
		})
	})
}

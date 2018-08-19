package epoch

import "testing"

func BenchmarkEpoch(b *testing.B) {
	b.Run("Protect+Unprotect", func(b *testing.B) {
		h := AcquireHandle()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			Protect(h)
			Unprotect(h)
		}
	})

	b.Run("Acquire+Release Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			h := AcquireHandle()
			for pb.Next() {
				Protect(h)
				Unprotect(h)
			}
		})
	})
}

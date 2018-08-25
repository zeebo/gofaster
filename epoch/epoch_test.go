package epoch

import "testing"

func BenchmarkEpoch(b *testing.B) {
	b.Run("Protect+Unprotect", func(b *testing.B) {
		b.ReportAllocs()

		h := AcquireHandle()
		defer ReleaseHandle(h)

		for i := 0; i < b.N; i++ {
			Protect(h)
			Unprotect(h)
		}
	})

	b.Run("Protect+Unprotect Parallel", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			h := AcquireHandle()
			defer ReleaseHandle(h)

			for pb.Next() {
				Protect(h)
				Unprotect(h)
			}
		})
	})
}

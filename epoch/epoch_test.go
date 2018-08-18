package epoch

import "testing"

func TestEpoch(t *testing.T) {
	assertDistinct := func(hs ...*Handle) {
		used := make(map[*Handle]int, len(hs))
		for i, h := range hs {
			if o, ok := used[h]; ok {
				t.Fatalf("handle index %d used already at index %d", i, o)
			}
			used[h] = i
		}
	}

	t.Run("Distinct", func(t *testing.T) {
		e := New()

		h1, h2, h3 := e.Acquire(), e.Acquire(), e.Acquire()
		assertDistinct(h1, h2, h3)
	})
}

func BenchmarkEpoch(b *testing.B) {
	b.Run("Acquire+Release", func(b *testing.B) {
		e := New()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			h := e.Acquire()
			e.Release(h)
		}
	})

	b.Run("Acquire+Release Parallel", func(b *testing.B) {
		e := New()

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				h := e.Acquire()
				e.Release(h)
			}
		})
	})
}

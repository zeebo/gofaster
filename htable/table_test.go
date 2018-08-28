package htable

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/machine"
	"github.com/zeebo/gofaster/internal/pcg"
)

func TestTable(t *testing.T) {
	t.SkipNow()

	h := epoch.AcquireHandle()
	defer epoch.ReleaseHandle(h)

	const max = 10

	table := New(4)
	for i := 0; i < max; i++ {
		data := []byte(fmt.Sprint(i))
		table.Insert(h, data, data)
	}

	for i := 0; i < max; i++ {
		data := []byte(fmt.Sprint(i))
		fmt.Println(string(table.Lookup(h, data)))
	}

	for i := 0; i < max; i++ {
		data := []byte(fmt.Sprint(i))
		fmt.Println(table.Delete(h, data))
	}

	for i := 0; i < max; i++ {
		data := []byte(fmt.Sprint(i))
		fmt.Println(string(table.Lookup(h, data)))
	}

	fmt.Printf("%#v\n", table)
}

func BenchmarkTable(b *testing.B) {
	var (
		dataS [256]string
		dataB [256][]byte
	)
	for i := 0; i < 256; i++ {
		dataS[i] = fmt.Sprint(i)
		dataB[i] = []byte(fmt.Sprint(i))
	}

	b.Run("Insert+Read+Delete Table", func(b *testing.B) {
		h := epoch.AcquireHandle()
		defer epoch.ReleaseHandle(h)
		table := New(4)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			table.Insert(h, dataB[i&255], dataB[i&255])
			table.Lookup(h, dataB[i&255])
			table.Delete(h, dataB[i&255])
		}
	})

	b.Run("Insert+Read+Delete Map", func(b *testing.B) {
		table := make(map[string][]byte)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			table[dataS[i&255]] = dataB[i&255]
			_ = table[dataS[i&255]]
			delete(table, dataS[i&255])
		}
	})

	b.Run("Insert+Read+Delete SyncMap", func(b *testing.B) {
		var table sync.Map

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			table.Store(dataS[i&255], dataB[i&255])
			table.Load(dataS[i&255])
			table.Delete(dataS[i&255])
		}
	})

	const (
		ratioInsert = 1
		ratioLookup = 30
		ratioDelete = 1
		ratioTotal  = ratioInsert + ratioLookup + ratioDelete
	)

	const (
		actionInsert = iota
		actionLookup
		actionDelete
	)

	actions := make([]int, ratioTotal)
	for i := 0; i < ratioInsert; i++ {
		actions[i] = actionInsert
	}
	for i := 0; i < ratioLookup; i++ {
		actions[i+ratioInsert] = actionLookup
	}
	for i := 0; i < ratioDelete; i++ {
		actions[i+ratioInsert+ratioLookup] = actionDelete
	}

	b.Run("Par Insert+Read+Delete Table", func(b *testing.B) {
		index := uint64(0)
		hs := make([]epoch.Handle, machine.MaxThreads)
		for i := range hs {
			hs[i] = epoch.AcquireHandle()
			defer epoch.ReleaseHandle(hs[i])
		}

		table := New(4)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := atomic.AddUint64(&index, 1) - 1
			h := hs[i]
			p := pcg.New(i, uint64(time.Now().UnixNano()))

			for pb.Next() {
				n := p.Uint32() % uint32(len(dataB))
				switch actions[p.Uint32()%ratioTotal] {
				case actionInsert:
					table.Insert(h, dataB[n], dataB[n])
				case actionLookup:
					table.Lookup(h, dataB[n])
				case actionDelete:
					table.Delete(h, dataB[n])
				}
			}
		})
	})

	b.Run("Par Insert+Read+Delete Map", func(b *testing.B) {
		index := uint64(0)
		mu := new(sync.RWMutex)
		table := make(map[string][]byte)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := atomic.AddUint64(&index, 1) - 1
			p := pcg.New(i, uint64(time.Now().UnixNano()))

			for pb.Next() {
				n := p.Uint32() % uint32(len(dataB))
				switch actions[p.Uint32()%ratioTotal] {
				case actionInsert:
					mu.Lock()
					table[dataS[n]] = dataB[n]
					mu.Unlock()
				case actionLookup:
					mu.RLock()
					_ = table[dataS[n]]
					mu.RUnlock()
				case actionDelete:
					mu.Lock()
					delete(table, dataS[n])
					mu.Unlock()
				}
			}
		})
	})

	b.Run("Par Insert+Read+Delete SyncMap", func(b *testing.B) {
		index := uint64(0)
		var table sync.Map

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := atomic.AddUint64(&index, 1) - 1
			p := pcg.New(i, uint64(time.Now().UnixNano()))

			for pb.Next() {
				n := p.Uint32() % uint32(len(dataB))
				switch actions[p.Uint32()%ratioTotal] {
				case actionInsert:
					table.Store(dataS[n], dataB[n])
				case actionLookup:
					table.Load(dataS[n])
				case actionDelete:
					table.Delete(dataS[n])
				}
			}
		})
	})
}

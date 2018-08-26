package htable

import (
	"unsafe"

	"github.com/zeebo/gofaster/internal/machine"
)

const (
	bucketSize = unsafe.Sizeof(bucket{})

	tagBits = 15
	tagMask = 1<<tagBits - 1
)

type bucket struct {
	entries  [7]uint64
	overflow *bucket
}

type ( // ensure the bucket is sized to a cache line
	_ [bucketSize - machine.CacheLine]byte
	_ [machine.CacheLine - bucketSize]byte
)

func (b *bucket) Delete(hash uint64, key []byte) bool {
	// tag := uint16(hash & tagMask)
	// for i := range &b.entries {
	// addr := &b.entries[i]
	// }
	return false
}

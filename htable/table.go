package htable

import (
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/zeebo/gofaster/internal/risky"
)

type Table struct {
	buckets []bucket
	bits    uint64 // 2^bits buckets
	mask    uint64
}

func (t *Table) index(i uint64) *bucket {
	ptr := risky.Index(unsafe.Pointer(&t.buckets), bucketSize, uintptr(i))
	return (*bucket)(unsafe.Pointer(ptr))
}

func (t *Table) Delete(key []byte) bool {
	hash := xxhash.Sum64(key)
	hash, idx := hash>>t.bits, hash&t.mask
	bucket := t.index(idx)
	return bucket.Delete(hash, key)
}

func (t *Table) Insert(key, value []byte) {

}

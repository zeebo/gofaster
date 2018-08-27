package htable

import (
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/risky"
)

const (
	tagBits = 15
	tagMask = 1<<tagBits - 1
)

type Table struct {
	buckets []bucket
	bits    uint64 // 2^bits buckets
	mask    uint64
}

// split turns the hash into a tag and bucket index.
func (t *Table) split(hash uint64) (tag uint16, idx uint64) {
	return uint16(hash & tagMask), (hash >> tagBits) & t.mask
}

// index returns the bucket for the given index.
func (t *Table) index(i uint64) *bucket {
	ptr := risky.Index(unsafe.Pointer(&t.buckets), bucketSize, uintptr(i))
	return (*bucket)(unsafe.Pointer(ptr))
}

// Delete removes the key from the table and returns true if it was able to.
func (t *Table) Delete(h epoch.Handle, key []byte) bool {
	tag, idx := t.split(xxhash.Sum64(key))
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		if found, deleted := bucket.Delete(h, tag, key); found {
			return deleted
		}
	}
	return false
}

// Lookup finds the value for the key, returning nil if no key matches.
func (t *Table) Lookup(h epoch.Handle, key []byte) []byte {
	tag, idx := t.split(xxhash.Sum64(key))
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		if found, val := bucket.Lookup(h, tag, key); found {
			return val
		}
	}
	return nil
}

func (t *Table) Insert(key, value []byte) {

}

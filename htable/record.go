package htable

import (
	"unsafe"

	"github.com/zeebo/gofaster/internal/risky"
)

// record keeps track of a key value pair with some metadata, where the key and
// value are allocated directly after the metadata.
type record struct {
	key uint64
	val uint64
	// key and value data follows directly in memory
}

const recordSize = unsafe.Sizeof(record{})

type ( // make sure the alignment is what we expect for alloc
	_ [8 - unsafe.Alignof(record{})]byte
	_ [unsafe.Alignof(record{}) - 8]byte
)

// newRecord constructs a record with the key and value directly next to each other
// in memory.
func newRecord(key, val []byte) *record {
	buf := risky.Alloc8(int(recordSize) + len(key) + len(val))

	// relies on the data pointer being first in a slice
	rec := *(**record)(unsafe.Pointer(&buf))
	rec.key = uint64(len(key))
	rec.val = uint64(len(val))

	copy(rec.Key(), key)
	copy(rec.Val(), val)

	return rec
}

// slice returns a byte starting offset bytes past the record, with the given length.
func (r *record) slice(offset uintptr, length int) []byte {
	return risky.Slice(unsafe.Pointer(uintptr(unsafe.Pointer(r))+offset), length)
}

// Key returns a byte slice containing the key in the record.
func (r *record) Key() []byte {
	return r.slice(recordSize, int(r.key))
}

// Val returns a byte slice containing the value in the record.
func (r *record) Val() []byte {
	return r.slice(recordSize+uintptr(r.key), int(r.val))
}

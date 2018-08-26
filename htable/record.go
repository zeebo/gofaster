package htable

import (
	"unsafe"

	"github.com/zeebo/gofaster/internal/machine"
)

type record struct {
	key uint64
	val uint64
	// key and value data follows directly in memory
}

const recordSize = 16 // we use an untyped constant on purpose

type ( // assert that our untyped constant matches the actual size
	_ [recordSize - unsafe.Sizeof(record{})]byte
	_ [unsafe.Sizeof(record{}) - recordSize]byte
)

func newRecord(key, val []byte) (rec *record) {
	buf := make([]byte, recordSize+len(key)+len(val))

	rec = (*record)(unsafe.Pointer(&buf[0]))
	rec.key = uint64(len(key))
	rec.val = uint64(len(val))

	copy(rec.Key(), key)
	copy(rec.Val(), val)

	return rec
}

func (r *record) indexBytes(offset uintptr) *[machine.MaxSlice]byte {
	return (*[machine.MaxSlice]byte)(unsafe.Pointer(uintptr(unsafe.Pointer(r)) + offset))
}

func (r *record) keyOffset() uintptr { return recordSize }
func (r *record) valOffset() uintptr { return recordSize + uintptr(r.key) }

// TODO(jeff): remove bounds checks on these calls
func (r *record) Key() []byte { return r.indexBytes(r.keyOffset())[:r.key:r.key] }
func (r *record) Val() []byte { return r.indexBytes(r.valOffset())[:r.val:r.val] }

package pin

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/gofaster/internal/machine"
)

// Location is an abstract value returned by Pin that can be used to Unpin the
// memory. It allows storing 16 bits of extra data, and provides atomic
// operations on a uint64.
type Location struct{ x uint64 }

func (l Location) String() string {
	return fmt.Sprintf("{id:%d index:%-2d extra:%04x}", l.id(), l.index(), l.Extra())
}

// newLocation constructs a location that helps find some pointer.
func newLocation(id uint32, index uint32) Location {
	return Location{uint64(index)<<(machine.MaxThreadBits+1) | uint64(id)<<1 | 1}
}

// id returns the encoded handle id inside of the location.
func (l Location) id() uint32 {
	return uint32(l.x>>1) & (1<<machine.MaxThreadBits - 1)
}

// index returns the index into the buffer of the location.
func (l Location) index() uint32 {
	return uint32(l.x) >> (machine.MaxThreadBits + 1)
}

// Nil returns if the location is conceptually nil.
func (l Location) Nil() bool { return l.x&1 == 0 }

// WithExtra returns an equivalent location with the associated data. It
// counts as the same abstract location, but will not compare equal.
func (l Location) WithExtra(data uint16) Location {
	return Location{uint64(l.x) | uint64(data)<<48}
}

// Extra returns the associated 16 extra bits of data.
func (l Location) Extra() uint16 {
	return uint16(uint64(l.x) >> 48)
}

// LoadLocation atomically loads the location from the address.
func LoadLocation(addr *Location) Location {
	return Location{atomic.LoadUint64(
		(*uint64)(unsafe.Pointer(addr)),
	)}
}

// StoreLocation atomically stores the location into the address.
func StoreLocation(addr *Location, val Location) {
	atomic.StoreUint64(
		(*uint64)(unsafe.Pointer(addr)),
		uint64(val.x),
	)
}

// CompareAndSwapLocation atomically performs a CAS operation on Locations.
func CompareAndSwapLocation(addr *Location, old, new Location) bool {
	return atomic.CompareAndSwapUint64(
		(*uint64)(unsafe.Pointer(addr)),
		uint64(old.x),
		uint64(new.x),
	)
}

package pin

import (
	"sync/atomic"

	"github.com/zeebo/gofaster/internal/machine"
)

// Location is an abstract value returned by Pin that can be used to Unpin the
// memory. It allows storing 16 bits of extra data, and provides atomic
// operations on a uint64.
type Location uint64

// LoadLocation atomically loads the location from the address
func LoadLocation(addr *uint64) Location {
	return Location(atomic.LoadUint64(addr))
}

// newLocation constructs a location that helps find some pointer.
func newLocation(id uint32, index uint32) Location {
	return Location(uint64(index)<<machine.MaxThreadBits | uint64(id))
}

// id returns the encoded handle id inside of the location.
func (l Location) id() uint32 {
	return uint32(l) & (1<<machine.MaxThreadBits - 1)
}

// index returns the index into the buffer of the location.
func (l Location) index() uint32 {
	return uint32(l) >> machine.MaxThreadBits
}

// WithExtra returns an equivalent location with the associated data. It
// counts as the same abstract location, but will not compare equal.
func (l Location) WithExtra(data uint16) Location {
	return Location(uint64(l) | uint64(data)<<48)
}

// Extra returns the associated 16 extra bits of data.
func (l Location) Extra() uint16 {
	return uint16(uint64(l) >> 48)
}

// Store atomically stores the location into the address.
func (l Location) Store(addr *uint64) {
	atomic.StoreUint64(addr, uint64(l))
}

// CAS atomically stores the location into the addr if it currently stores old.
func (l Location) CAS(addr *uint64, old Location) bool {
	return atomic.CompareAndSwapUint64(addr, uint64(old), uint64(l))
}

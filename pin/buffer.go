package pin

import (
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/gofaster/internal/debug"
	"github.com/zeebo/gofaster/internal/machine"
)

const (
	ptrSize = unsafe.Sizeof(uintptr(0))
)

// buffer keeps track of pinned items per thread.
type buffer struct {
	unpinned unsafe.Pointer   // linked list of unpinned locations
	data     []unsafe.Pointer // buffer of pinned items
	start    uint64           // start index into data for adding.
	free     uint64           // amount free, used for resizing.
	mask     uint64           // mask for modulo indexing into data
	bits     uint64           // number of bits in the mask
}

type ( // ensure the buffer is sized to a cache line
	_ [unsafe.Sizeof(buffer{}) - machine.CacheLine]byte
	_ [machine.CacheLine - unsafe.Sizeof(buffer{})]byte
)

// grow doubles the buffer's size
func (b *buffer) grow() {
	b.free += uint64(len(b.data))
	b.mask = b.mask<<1 | 1
	b.bits++

	next := make([]unsafe.Pointer, 2*len(b.data))
	copy(next, b.data)
	b.data = next
}

// index returns the address of the element at the given index modulo the mask.
func (b *buffer) index(i uint64) *unsafe.Pointer {
	debug.Assert("index out of range", func() bool {
		return i&b.mask < uint64(len(b.data))
	})

	// relies on the data pointer being first in a slice
	ptr := unsafe.Pointer(
		uintptr(*(*unsafe.Pointer)(unsafe.Pointer(&b.data))) +
			ptrSize*uintptr(i&b.mask))
	return (*unsafe.Pointer)(ptr)
}

// unpin removes the location from the data, and increments free.
func (b *buffer) unpin(loc Location) {
	*b.index(loc.index()) = nil
	b.free++
}

// pin adds the pointer to the location and decrements free.
func (b *buffer) pin(loc Location, ptr unsafe.Pointer) {
	*b.index(loc.index()) = ptr
	b.free--
}

// unpinnedElement is a linked list for tracking cross thread unpin calls.
// if the length becomes too long, a Pin call drains it.
type unpinnedElement struct {
	next unsafe.Pointer
	loc  Location
}

// getUnpinned reads and clears the unpinned linked list.
func (b *buffer) getUnpinned() unsafe.Pointer {
retry:
	current := atomic.LoadPointer(&b.unpinned)
	if current == nil {
		return nil
	}
	if !atomic.CompareAndSwapPointer(&b.unpinned, current, nil) {
		goto retry
	}
	return current
}

// addUnpinned adds a location to the head of the unpinned linked list.
func (b *buffer) addUnpinned(loc Location) {
	element := &unpinnedElement{loc: loc}
retry:
	current := atomic.LoadPointer(&b.unpinned)
	element.next = current
	if !atomic.CompareAndSwapPointer(&b.unpinned, current, unsafe.Pointer(element)) {
		goto retry
	}
}

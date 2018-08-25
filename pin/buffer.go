package pin

import (
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/gofaster/internal/machine"
)

const (
	ptrSize = unsafe.Sizeof(uintptr(0))
)

// buffer keeps track of pinned items per thread.
type buffer struct {
	// linked list of unpinned locations. atomic/concurrent
	unpinned unsafe.Pointer

	// the rest of the fields are "thread local"
	data  []unsafe.Pointer // buffer of pinned items
	start uint32           // start index into data for adding.
	free  uint32           // amount free, used for resizing.
	mask  uint32           // mask for modulo indexing into data
	bits  uint32           // number of bits in the mask

	_ [16]byte
}

type ( // ensure the buffer is sized to a cache line
	_ [unsafe.Sizeof(buffer{}) - machine.CacheLine]byte
	_ [machine.CacheLine - unsafe.Sizeof(buffer{})]byte
)

// newBuffer allocates a buffer with spaces for 2^bits pointers.
func newBuffer(bits uint32) buffer {
	var b buffer
	b.data = make([]unsafe.Pointer, 1<<bits)
	b.free = 1 << bits
	b.mask = 1<<bits - 1
	b.bits = bits
	return b
}

// grow doubles the buffer's size
func (b *buffer) grow() {
	b.free += uint32(len(b.data))
	b.mask = b.mask<<1 | 1
	b.bits++

	next := make([]unsafe.Pointer, 2*len(b.data))
	copy(next, b.data)
	b.data = next
}

// index returns the address of the element at the given index modulo the mask.
func (b *buffer) index(i uint32) *unsafe.Pointer {
	// relies on the data pointer being first in a slice
	data := *(*unsafe.Pointer)(unsafe.Pointer(&b.data))
	ptr := unsafe.Pointer(uintptr(data) + ptrSize*uintptr(i))
	return (*unsafe.Pointer)(ptr)
}

// pin adds the pointer to the location and decrements free.
func (b *buffer) pin(loc Location, ptr unsafe.Pointer) {
	*b.index(loc.index()) = ptr
	b.free--
}

// unpin removes the location from the data, and increments free.
func (b *buffer) unpin(loc Location) {
	*b.index(loc.index()) = nil
	b.free++
}

// read returns the value of the pointer identified by the location.
func (b *buffer) read(loc Location) unsafe.Pointer {
	return *b.index(loc.index())
}

// unpinnedElement is a linked list for tracking cross thread unpin calls.
// if the length becomes too long, a Pin call drains it.
type unpinnedElement struct {
	next unsafe.Pointer
	loc  Location
}

// consumeUnpinned reads and clears the unpinned linked list.
func (b *buffer) consumeUnpinned() unsafe.Pointer {
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

// appendUnpinned adds a location to the head of the unpinned linked list.
func (b *buffer) appendUnpinned(loc Location) {
	element := &unpinnedElement{loc: loc}
retry:
	current := atomic.LoadPointer(&b.unpinned)
	element.next = current
	if !atomic.CompareAndSwapPointer(&b.unpinned, current, unsafe.Pointer(element)) {
		goto retry
	}
}

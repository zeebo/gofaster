package pin

import (
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/machine"
)

// pinnedData keeps track of all of the buffers
var pinnedData struct {
	buffers [machine.MaxThreads]buffer
}

// allocate the buffers with hopefully enough space
func init() {
	const bits = 10
	for i := range &pinnedData.buffers {
		buffer := &pinnedData.buffers[i]
		buffer.data = make([]unsafe.Pointer, 1<<bits)
		buffer.free = 1 << bits
		buffer.mask = 1<<bits - 1
		buffer.bits = bits
	}
}

// getBuffer returns the buffer associated to the given handle id.
func getBuffer(id uint32) *buffer {
	return &pinnedData.buffers[id%machine.MaxThreads]
}

// Location is an abstract value returned by Pin that can be used to Unpin the
// memory.
type Location uint64

// newLocation constructs a location that helps find some pointer.
func newLocation(id uint32, index uint64) Location {
	return Location(index<<machine.MaxThreadBits | uint64(id))
}

// id returns the encoded handle id inside of the location.
func (l Location) id() uint32 {
	return uint32(l) & (1<<machine.MaxThreadBits - 1)
}

// index returns the index into the buffer of the location.
func (l Location) index() uint64 {
	return uint64(l) >> machine.MaxThreadBits
}

// Pin ensures the pointer will not be garbage collected until Unpin is called
// on the returned Location. It is not safe to use concurrently with the same
// Handle.
func Pin(h epoch.Handle, ptr unsafe.Pointer) Location {
	buffer := getBuffer(h.Id())

	// acquire and process any unpinned linked list items
	unpinned := buffer.getUnpinned()
	for unpinned != nil {
		element := (*unpinnedElement)(unpinned)
		buffer.unpin(element.loc)
		unpinned = element.next
	}

	// TODO(jeff): handle buffer shrinking :)
	if buffer.free == 0 {
		buffer.grow()
	}

	start := buffer.start
	end := buffer.start + uint64(len(buffer.data))

	for start < end {
		if *buffer.index(start) == nil {
			loc := newLocation(h.Id(), start&buffer.mask)
			buffer.pin(loc, ptr)
			buffer.start = start + 1
			return loc
		}
		start++
	}

	panic("impossible: buffer full")
}

// Unpin allows the pointer for the returned Location to be garbage collected.
// It is undefined if called multiple times on the same Location, and it is
// not safe to use concurrently with the same Handle.
func Unpin(h epoch.Handle, loc Location) {
	id := h.Id()
	buffer := getBuffer(id)

	if id == loc.id() {
		buffer.unpin(loc)
	} else {
		buffer.addUnpinned(loc)
	}
}

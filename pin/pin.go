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
	const bits = 4
	for i := range &pinnedData.buffers {
		pinnedData.buffers[i] = newBuffer(bits)
	}
}

// getBuffer returns the buffer associated to the given handle id.
func getBuffer(id uint32) *buffer {
	return &pinnedData.buffers[id%machine.MaxThreads]
}

// Pin ensures the pointer will not be garbage collected until Unpin is called
// on the returned Location. It is not safe to use concurrently with the same
// Handle.
func Pin(h epoch.Handle, ptr unsafe.Pointer) Location {
	buffer := getBuffer(h.Id())

	// acquire and process any unpinned linked list items
	unpinned := buffer.consumeUnpinned()
	for unpinned != nil {
		element := (*unpinnedElement)(unpinned)
		buffer.unpin(element.loc)
		unpinned = element.next
	}

	// TODO(jeff): handle buffer shrinking :)
	// tricky because there could be locations in the upper half.
	if buffer.free == 0 {
		buffer.grow()
	}

	start := buffer.start & buffer.mask
	end := buffer.start + uint32(len(buffer.data))

	for start < end {
		if *buffer.index(start) == nil {
			loc := newLocation(h.Id(), start&buffer.mask)
			buffer.pin(loc, ptr)
			buffer.start++
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
	id := loc.id()
	buffer := getBuffer(id)

	if id == h.Id() {
		buffer.unpin(loc)
	} else {
		buffer.appendUnpinned(loc)
	}
}

// Read reads the pointer stored by the location. It does not require any handle,
// can can be called concurrently with itself, but not with or after Unpin for the
// location.
func Read(loc Location) unsafe.Pointer {
	return getBuffer(loc.id()).read(loc)
}

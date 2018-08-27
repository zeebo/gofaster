package htable

import (
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/machine"
	"github.com/zeebo/gofaster/pin"
)

// freelistEntry is a doubly linked list of locations.
type freelistEntry struct {
	next *freelistEntry
	prev *freelistEntry
	loc  pin.Location
}

// Remove takes the entry out of the linked list.
func (f *freelistEntry) Remove() {
	f.prev.next = f.next
	f.next.prev = f.prev
}

// Prepend allocates and returns a new freelistEntry before this one.
func (f *freelistEntry) Prepend() *freelistEntry {
	out := new(freelistEntry)
	out.next, f.prev = f, out
	return out
}

// freelistHead is a cache line sized struct for keeping track of a
// free list per thread.
type freelistHead struct {
	entry *freelistEntry
	_     [56]byte
}

type ( // make sure freelistHead is cache line sized
	_ [machine.CacheLine - unsafe.Sizeof(freelistHead{})]byte
	_ [unsafe.Sizeof(freelistHead{}) - machine.CacheLine]byte
)

// freelistData keeps track of a freelist of locations for each thread.
var freelistData struct {
	heads [machine.MaxThreads]freelistHead
}

// addFreelist adds the location to the freelist for the thread local storage
// for the handle.
func addFreelist(h epoch.Handle, loc pin.Location) {
	head := &freelistData.heads[h.Id()%machine.MaxThreads]
	head.entry = head.entry.Prepend()
	head.entry.loc = loc
}

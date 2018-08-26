// package risky provides unsafe helpers.
package risky

import (
	"unsafe"
)

// Index returns the address to the element in the slice at the slot, given each
// element is size bytes.
func Index(slice unsafe.Pointer, size, slot uintptr) *unsafe.Pointer {
	// relies on the data pointer being first in a slice
	data := *(*unsafe.Pointer)(slice)
	ptr := unsafe.Pointer(uintptr(data) + size*slot)
	return (*unsafe.Pointer)(ptr)
}

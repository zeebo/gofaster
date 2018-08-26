// package risky provides unsafe helpers.
package risky

import (
	"reflect"
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

// Slice returns a []byte with the given length using the data pointer.
func Slice(data unsafe.Pointer, length int) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(data),
		Len:  length,
		Cap:  length,
	}))
}

// Reslice returns a []byte with the given length for the slice.
func Reslice(slice unsafe.Pointer, length int) []byte {
	hdr := (*reflect.SliceHeader)(slice)
	hdr.Len = length
	hdr.Cap = length
	return *(*[]byte)(unsafe.Pointer(hdr))
}

// Alloc8 creates a []byte that is 64 bit aligned by using a []uint64 as a backing slice.
func Alloc8(bytes int) []byte {
	data := make([]uint64, (bytes+7)/8)
	return Reslice(unsafe.Pointer(&data), bytes)
}

// Alloc4 creates a []byte that is 32 bit aligned by using a []uint32 as a backing slice.
func Alloc4(bytes int) []byte {
	data := make([]uint32, (bytes+3)/4)
	return Reslice(unsafe.Pointer(&data), bytes)
}

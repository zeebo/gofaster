package epoch

import (
	"sync/atomic"

	"github.com/zeebo/gofaster/internal/machine"
)

var handleData struct {
	next uint32
	used [machine.MaxThreads]uint32
}

// Handle represents a thread handle. It should not cross threads for maximum performance. Calls
// involving the same Handle must not happen concurrently.
type Handle struct {
	id uint32
}

// AcquireHandle acquires a unique Handle for the thread.
func AcquireHandle() Handle {
	start := atomic.AddUint32(&handleData.next, 1)
	end := start + machine.MaxThreads*2

retry:
	if start == end {
		panic("too many thread handles")
	}
	id := start % machine.MaxThreads

	if !atomic.CompareAndSwapUint32(&handleData.used[id], 0, 1) {
		start++
		goto retry
	}

	return Handle{id: id}
}

// ReleaseHandle releases the handle for the thread, letting it be used by other threads.
func ReleaseHandle(h Handle) {
	atomic.StoreUint32(&handleData.used[h.id%machine.MaxThreads], 0)
}

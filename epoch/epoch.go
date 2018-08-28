package epoch

import (
	"runtime"
	"sync/atomic"

	"github.com/zeebo/gofaster/internal/machine"
)

// TODO(jeff): we use atomic loads everywhere when they might not be totally
// necessary. can they be relaxed, and will the race detector complain?

const (
	epochMaxTriggers = 256
)

var epochData struct {
	// keep track of the current epoch
	current uint64
	_       [56]uint8

	// keep track of which epoch is safe
	safe uint64
	_    [56]uint8

	// keep track of entries
	entries [machine.MaxThreads]entry

	// keep track of triggers
	trigger_count uint64
	_             [56]uint8
	triggers      [epochMaxTriggers]trigger
}

func init() {
	epochData.current = 1
	for i := range &epochData.triggers {
		epochData.triggers[i] = newTrigger()
	}
}

// getEntry returns the entry bound to the given handle id.
func getEntry(id uint32) *entry {
	id *= 8
	id += id / machine.MaxThreads
	return &epochData.entries[id%machine.MaxThreads]
}

// Protect enters the protected region of the epoch. It may be called multiple times with only
// one Unprotect necessary.
func Protect(h Handle) uint64 {
	entry := getEntry(h.Id())
	current := atomic.LoadUint64(&epochData.current)
	atomic.StoreUint64(&entry.local, current)
	return current
}

// ProtectAndDrain enters the protected region of the epoch, draining any triggers if possible.
// It may be called multiple times with only one Unprotect necessary.
func ProtectAndDrain(h Handle) uint64 {
	epoch := Protect(h)
	if atomic.LoadUint64(&epochData.trigger_count) > 0 {
		Drain(h, epoch)
	}
	return epoch
}

// IsProtected returns if the handle is in the protected region.
func IsProtected(h Handle) bool {
	entry := getEntry(h.Id())
	return atomic.LoadUint64(&entry.local) != 0
}

// LocalEpoch returns the local epoch for the handle.
func LocalEpoch(h Handle) uint64 {
	entry := getEntry(h.Id())
	return atomic.LoadUint64(&entry.local)
}

// Unprotect exits the protected region.
func Unprotect(h Handle) {
	entry := getEntry(h.Id())
	atomic.StoreUint64(&entry.local, 0)
}

// Drain runs any triggers that are safe to run. The provided epoch is used as an
// initial epoch for computing which epoch is safe.
func Drain(h Handle, epoch uint64) {
	safe := ComputeSafe(epoch)

	for i := range &epochData.triggers {
		trigger := &epochData.triggers[i]
		epoch := trigger.Epoch()
		if epoch <= safe && trigger.Run(h, epoch) {
			if atomic.AddUint64(&epochData.trigger_count, ^uint64(0)) == 0 {
				break
			}
		}
	}
}

// Bump increments the global epoch, draining any triggers that can be drained.
func Bump(h Handle) uint64 {
	epoch := atomic.AddUint64(&epochData.current, 1)
	if atomic.LoadUint64(&epochData.trigger_count) > 0 {
		Drain(h, epoch)
	}
	return epoch
}

// BumpWith increments the global epoch and adds the action into the trigger queue.
func BumpWith(h Handle, action func(Handle)) uint64 {
retry:
	prior := Bump(h) - 1
	failures := 0

finished:
	for {
		for i := range &epochData.triggers {
			trigger := &epochData.triggers[i]
			epoch := trigger.Epoch()
			safe := atomic.LoadUint64(&epochData.safe)

			if epoch == triggerFree && trigger.Store(prior, action) {
				atomic.AddUint64(&epochData.trigger_count, 1)
				break finished
			}

			if epoch <= safe && trigger.Swap(h, epoch, prior, action) {
				break finished
			}
		}

		failures++
		if failures == 500 {
			ComputeSafe(Protect(h))
			runtime.Gosched()
			goto retry
		}
	}

	return prior + 1
}

// ComputeSafe finds the current safe epoch across all the entries, using the
// provided epoch as an initial value.
func ComputeSafe(epoch uint64) uint64 {
	safe := atomic.LoadUint64(&epochData.safe)

	oldest := epoch
	for i := range &epochData.entries {
		local := atomic.LoadUint64(&epochData.entries[i].local)
		if local != 0 && local < oldest {
			oldest = local
		}
	}
	oldest--

	if oldest <= safe || !atomic.CompareAndSwapUint64(&epochData.safe, safe, oldest) {
		return safe
	}

	return oldest
}

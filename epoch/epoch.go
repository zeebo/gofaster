package epoch

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/zeebo/gofaster/machine"
)

const (
	epochDrainEntries = 256
)

// Epoch keeps track of
type Epoch struct {
	// keep track of the current epoch
	current uint64
	_       machine.Pad56

	// keep track of which epoch is safe
	safe uint64
	_    machine.Pad56

	// keep track of thread handles
	tnext    uint64
	_        machine.Pad56
	thandles [machine.MaxThreads]Handle

	// keep track of triggers
	dcount    uint64
	_         machine.Pad56
	dtriggers [epochDrainEntries]Trigger
}

// New returns a new Epoch.
func New() *Epoch {
	e := &Epoch{
		current: 1,
		safe:    0,
	}
	for i := range &e.thandles {
		e.thandles[i].phase = phaseRest
	}
	for i := range &e.dtriggers {
		e.dtriggers[i].epoch = triggerFree
	}
	return e
}

// Acquire returns a Handle for interacting with the Epoch.
func (e *Epoch) Acquire() *Handle {
	current := atomic.LoadUint64(&e.current)
	start := atomic.AddUint64(&e.tnext, 1)
	end := start + machine.MaxThreads*2

retry:
	if start == end {
		return nil
	}
	index := start % machine.MaxThreads

	if !atomic.CompareAndSwapUint64(&e.thandles[index].local, 0, current) {
		start++
		goto retry
	}

	return &e.thandles[index]
}

// AcquireAndDrain returns a Handle for interacting with the Epoch and
// processes entries in the drain list if possible.
func (e *Epoch) AcquireAndDrain() *Handle {
	handle := e.Acquire()
	if atomic.LoadUint64(&e.dcount) > 0 {
		e.Drain(handle.local)
	}
	return handle
}

// Release returns the Handle back to the Epoch so that it may be reused.
func (e *Epoch) Release(h *Handle) {
	atomic.StoreUint64(&h.local, 0)
}

// Drain processes any triggers that are set to run at or before the
// oldest safe epoch.
func (e *Epoch) Drain(epoch uint64) {
	e.ComputeSafe(epoch)

	for i := range &e.dtriggers {
		trigger := &e.dtriggers[i]
		epoch := trigger.Epoch()
		safe := atomic.LoadUint64(&e.safe)

		if epoch <= safe &&
			trigger.Run(epoch) &&
			atomic.AddUint64(&e.dcount, ^uint64(0)) == 0 {

			break
		}
	}

}

// ComputeSafe computes the current safe Epoch and returns it.
func (e *Epoch) ComputeSafe(epoch uint64) uint64 {
	oldest := epoch
	for i := range &e.thandles {
		local := atomic.LoadUint64(&e.thandles[i].local)
		if local != 0 && local < oldest {
			oldest = local
		}
	}
	atomic.StoreUint64(&e.safe, oldest-1)
	return oldest - 1
}

// Bump increments the current epoch, draining any Triggers.
func (e *Epoch) Bump() uint64 {
	epoch := atomic.AddUint64(&e.current, 1)
	if atomic.LoadUint64(&e.dcount) > 0 {
		e.Drain(epoch)
	}
	return epoch
}

// BumpWith increments the current epoch, adding the provided action
// as a Trigger to run.
func (e *Epoch) BumpWith(action func()) uint64 {
	prior := e.Bump() - 1
	failures := 0

finished:
	for {
		for i := range &e.dtriggers {
			trigger := &e.dtriggers[i]
			epoch := trigger.Epoch()

			if epoch == triggerFree && trigger.Store(epoch, action) {
				break finished
			}

			safe := atomic.LoadUint64(&e.safe)
			if epoch <= safe && trigger.Swap(epoch, prior, action) {
				break finished
			}
		}

		failures++
		if failures == 500 {
			failures = 0
			fmt.Fprintln(os.Stderr, "Slowdown: Unable to add trigger to epoch")
			time.Sleep(time.Second)
		}
	}

	atomic.AddUint64(&e.dcount, 1)
	return prior + 1
}

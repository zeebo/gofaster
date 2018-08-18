package epoch

import (
	"unsafe"

	"github.com/zeebo/gofaster/machine"
)

const (
	phasePrepIndexCheckpoint uint32 = iota
	phaseIndexCheckpoint
	phasePrepare
	phaseInProgress
	phaseWaitPending
	phaseWaitFlush
	phaseRest
	phasePersistenceCallback
	phaseGCIOPending
	phaseGCInProgress
	phaseGrowPrepare
	phaseGrowInProgress
)

// Handle is required to interact with the Epoch system. It should be used from one
// goroutine at a time, hopefully on the same thread.
type Handle struct {
	local     uint64
	reentrant uint32
	phase     uint32 // TODO(jeff): typed constants
	_         machine.Pad48
}

type ( // assert same size as a cache line
	_ [machine.CacheLine - unsafe.Sizeof(Handle{})]byte
	_ [unsafe.Sizeof(Handle{}) - machine.CacheLine]byte
)

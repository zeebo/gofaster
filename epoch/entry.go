package epoch

import (
	"unsafe"

	"github.com/zeebo/gofaster/internal/machine"
)

const (
	phaseEmpty uint32 = iota
	phasePrepIndexCheckpoint
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

type entry struct {
	local uint64
	phase uint32
	_     [52]uint8
}

type ( // ensure entries are exactly the size of a cache line
	_ [unsafe.Sizeof(entry{}) - machine.CacheLine]byte
	_ [machine.CacheLine - unsafe.Sizeof(entry{})]byte
)

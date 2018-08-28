package machine

const (
	CacheLine     = 64
	MaxThreadBits = 6
	MaxThreads    = 1 << MaxThreadBits
	MaxSlice      = 1<<50 - 1
)

type ( // ensure MaxThreads is actually 64.
	_ [MaxThreads - 64]byte
	_ [64 - MaxThreads]byte
)

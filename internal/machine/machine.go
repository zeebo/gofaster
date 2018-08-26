package machine

const (
	CacheLine     = 64
	MaxThreadBits = 5
	MaxThreads    = 1 << MaxThreadBits
	MaxSlice      = 1<<50 - 1
)

type ( // ensure MaxThreads is actually 32.
	_ [MaxThreads - 32]byte
	_ [32 - MaxThreads]byte
)

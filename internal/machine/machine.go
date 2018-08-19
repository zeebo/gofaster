package machine

const (
	CacheLine  = 64
	MaxThreads = 32
)

type (
	Pad64 [64]uint8
	Pad56 [56]uint8
	Pad48 [48]uint8
	Pad40 [40]uint8
	Pad32 [32]uint8
	Pad24 [24]uint8
	Pad16 [16]uint8
	Pad8  [8]uint8
)

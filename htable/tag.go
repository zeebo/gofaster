package htable

const (
	tagTentativeBit = 1 << 15
	tagDeleteBit    = 1 << 14
	tagHashBits     = 14
	tagHashMask     = 1<<tagHashBits - 1
)

type tag uint16

func (t tag) Tentative() bool { return t&tagTentativeBit > 0 }
func (t tag) Deleting() bool  { return t&tagDeleteBit > 0 }

func (t tag) WithTentative() tag    { return t | tagTentativeBit }
func (t tag) WithoutTentative() tag { return t &^ tagTentativeBit }
func (t tag) WithDelete() tag       { return t | tagDeleteBit }
func (t tag) WithoutDelete() tag    { return t &^ tagDeleteBit }

func (t tag) Hash() uint16 { return uint16(t & tagHashMask) }

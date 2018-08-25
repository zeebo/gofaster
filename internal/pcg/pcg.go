package pcg

import (
	"math/bits"
)

type PCG struct {
	state uint64
	inc   uint64
}

const mul = 6364136223846793005

// New constructs a pcg with the given state and inc.
func New(state, inc uint64) PCG {
	// this code is equiv to initializing a pcg with a 0 state and the updated
	// inc and running
	//
	//    p.Uint32()
	//    p.state += state
	//    p.Uint32()
	//
	// to get the generator started
	inc = inc<<1 | 1
	return PCG{
		state: (inc+state)*mul + inc,
		inc:   inc,
	}
}

// Uint32 returns a random uint32.
func (p *PCG) Uint32() uint32 {
	// this branch will be predicted to be false in most cases and so is
	// essentially free. this causes the zero value of a pcg to be the same as
	// New(0, 0).
	if p.inc == 0 {
		*p = New(0, 0)
	}

	// update the state (LCG step)
	oldstate := p.state
	p.state = oldstate*mul + p.inc

	// apply the output permutation to the old state
	// NOTE: this should be a right rotate but i can't coerce the compiler into
	// doing it. since any rotate should be sufficient for the output compression
	// function, this ought to be fine, and is significantly faster.

	xorshift := uint32(((oldstate >> 18) ^ oldstate) >> 27)
	return bits.RotateLeft32(xorshift, int(oldstate>>59))
}

// Intn returns an int uniformly in [0, n)
func (p *PCG) Intn(n int) int {
	return fastMod(p.Uint32(), n)
}

// fastMod computes n % m assuming that n is a random number in the full
// uint32 range.
func fastMod(n uint32, m int) int {
	return int((uint64(n) * uint64(m)) >> 32)
}

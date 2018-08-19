// +build !release

package assert

func That(info string, fn func() bool) {
	if !fn() {
		panic("assertion failed: " + info)
	}
}

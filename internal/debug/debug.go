// +build !release

package debug

func Assert(info string, fn func() bool) {
	if !fn() {
		panic("assertion failed: " + info)
	}
}

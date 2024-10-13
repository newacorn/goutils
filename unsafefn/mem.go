package unsafefn

import "unsafe"

func B2S(bs []byte) (s string) {
	s = unsafe.String(unsafe.SliceData(bs), len(bs))
	return
}
func S2B(s string) (bs []byte) {
	bs = unsafe.Slice(unsafe.StringData(s), len(s))
	return
}

// Bytes allocates a byte slice but does not clean up the memory it references.
// Throw a fatal error instead of panic if cap is greater than runtime.maxAlloc.
// NOTE: MUST set any byte element before it's read.
func Bytes(len, cap int) (b []byte) {
	if len < 0 || len > cap {
		panic("dirtmake.Bytes: len out of range")
	}
	p := mallocgc(uintptr(cap), nil, false)
	sh := (*slice)(unsafe.Pointer(&b))
	sh.data = p
	sh.len = len
	sh.cap = cap
	return
}

type slice struct {
	data unsafe.Pointer
	len  int
	cap  int
}

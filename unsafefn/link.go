package unsafefn

import "unsafe"

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer

//go:noescape
//go:linkname NanoTime runtime.nanotime
func NanoTime() int64

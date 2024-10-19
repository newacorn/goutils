// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io

import (
	"errors"
	bpool "github.com/newacorn/simple-bytes-pool"
	"io"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ErrClosedPipe is the error used for read or write operations on a closed pipe.
var ErrClosedPipe = errors.New("io: read/write on closed pipe")

// WriterIoEOF The write end has finished writing the intended content and has closed the write end.
var WriterIoEOF = &writerError{io.EOF}

// ErrBufRecycled The underlying cache has been reclaimed, and read/write operations can no longer be performed.
var ErrBufRecycled = errors.New("buf had been recycled")

// closing a closed pipe from reader's and writer's close method.
var errReClosingClosedPipe = errors.New(`use of closed pipe`)

// when close pipe buf not empty; only read from read method
var errorClosingNotEmptyPipe = errors.New(`closing busy buffer`)

const defaultBufSize = 4096

// A pipe is the shared pipe structure underlying PipeReader and PipeWriter.
// When the read/write segment is less than 1/4 of the cache, efficiency is maximized.
// Half-duplex communication implementation.
type pipe struct {
	rerr    error
	werr    error
	errMux  sync.Mutex
	buf     []byte
	bufPtr  unsafe.Pointer
	r, w    int
	l, c    int
	wr      sync.Mutex
	done    chan struct{}
	notifyR chan struct{}
	notifyW chan struct{}
	close   atomic.Bool
}

// PipeWithSize Create read and write ends using a specified cache size.
func PipeWithSize(size int) (*PipeReader, *PipeWriter) {
	py := bpool.Get(size)
	py.B = py.B[:size]
	pw := &PipeWriter{r: PipeReader{pipe: pipe{
		notifyR: make(chan struct{}, 1),
		notifyW: make(chan struct{}, 1),
		done:    make(chan struct{}),
		c:       size,
		buf:     py.B,
		bufPtr:  unsafe.Pointer(&py.B[0]),
	}}}
	pw.r.bytes = []*bpool.Bytes{py}
	return &pw.r, pw
}

// Pipe Create read and write ends using the default cache size.
func Pipe() (*PipeReader, *PipeWriter) {
	py := bpool.Get(defaultBufSize)
	py.B = py.B[:defaultBufSize]
	pw := &PipeWriter{r: PipeReader{pipe: pipe{
		notifyR: make(chan struct{}, 1),
		notifyW: make(chan struct{}, 1),
		done:    make(chan struct{}),
		c:       defaultBufSize,
		buf:     py.B,
		bufPtr:  unsafe.Pointer(&py.B[0]),
	}}}
	pw.r.bytes = []*bpool.Bytes{py}
	return &pw.r, pw
}

// A PipeReader is the read end of a pipe.
type PipeReader struct {
	pipe
	bytes []*bpool.Bytes
}

// Read implements the standard Read interface:
// it reads data from the pipe, blocking until a writer
// arrives or the write end is closed.
// If the write end is closed with an error, that error is
// returned as err; otherwise err is EOF.
func (r *PipeReader) Read(data []byte) (n int, err error) {
	return r.pipe.read(data)
}

func (r *PipeReader) ReadUnsafe(data []byte) (n int, err error) {
	return r.pipe.readUnsafe(data)
}

func (r *PipeReader) ReadByte() (b byte, err error) {
	r.wr.Lock()
	if r.buf == nil {
		r.wr.Unlock()
		err = ErrBufRecycled
		return
	}
	if r.pipe.l > 0 {
		b = r.pipe.buf[r.pipe.r]
		r.pipe.l--
		r.pipe.r++
		r.wr.Unlock()
		return
	}
	r.wr.Unlock()
	p := [1]byte{0}
	_, err = r.pipe.read(p[:])
	return p[0], err
}

// CloseWithError closes the reader; subsequent writes
// to the write half of the pipe will return the error err.
// never overwrites the previous error if it exists.
func (r *PipeReader) CloseWithError(err error) (e error) {
	r.errMux.Lock()
	e = r.pipe.closeRead(err)
	r.errMux.Unlock()
	return
}

// Close closes the reader; subsequent writes return the error [ErrClosedPipe].
func (r *PipeReader) Close() error {
	return r.CloseWithError(nil)
}

// RecycleItems Reclaiming the underlying cache makes further read or write operations
// meaningless after this method is called.
func (r *PipeReader) RecycleItems() {
	_ = r.CloseWithError(ErrBufRecycled)
	r.wr.Lock()
	if r.buf == nil {
		r.wr.Unlock()
		return
	}
	r.buf = nil
	if len(r.bytes) > 0 {
		if r.bytes[0] != nil {
			r.bytes[0].RecycleToPool00()
			r.bytes = nil
		}
	}
	r.wr.Lock()
	return
}

// A PipeWriter is the write end of a pipe.
type PipeWriter struct{ r PipeReader }

// Write implements the standard Write interface:
// it writes data to the pipe, blocking until one or more readers
// have consumed all the data or the read end is closed.
// If the read end is closed with an error, that err is
// returned as err; otherwise err is [ErrClosedPipe].
func (w *PipeWriter) Write(data []byte) (n int, err error) {
	return w.r.pipe.write(data)
}

func (w *PipeWriter) WriteUnsafe(data []byte) (n int, err error) {
	return w.r.pipe.writeUnsafe(data)
}

func (w *PipeWriter) Flush() (err error) {
	return w.r.pipe.flush()
}

func (p *pipe) WriteTo(w io.Writer) (n int64, err error) {
start:
	if p.close.Load() {
		err = p.readCloseError()
		if //goland:noinspection GoTypeAssertionOnErrors
		_, ok := err.(*readerError); ok {
			return
		}

	}
	var n1 int
	p.wr.Lock()
	if p.buf == nil {
		err = ErrBufRecycled
		p.wr.Unlock()
		return
	}
	if p.l != 0 {
		err = nil
	} else {
		if //goland:noinspection GoDirectComparisonOfErrors
		err == WriterIoEOF {
			err = io.EOF
		}
		p.wr.Unlock()
		return
	}
	// write data from buf to w
	if p.l > 0 {
		// copy from buf
		if p.r < p.w {
			n1, err = w.Write(p.buf[p.r:p.w])
			p.r += n1
			n = int64(n1)
		} else {
			n1, err = w.Write(p.buf[p.r:])
			p.r += n1
			n = int64(n1)
			if err != nil {
				n1, err = w.Write(p.buf[:p.w])
				p.r = n1
				n += int64(n1)
			}
		}
		p.l -= int(n)
		if //goland:noinspection GoDirectComparisonOfErrors
		p.l == 0 && err == WriterIoEOF {
			err = io.EOF
		}
		p.wr.Unlock()
		notifyW(p)
		return
	}
	// no buf copy to w
	// wait sign from writer
	p.wr.Unlock()
	// nothing read wait sign from writer.
	select {
	case <-p.notifyR: // writer had wrote data
		goto start // retry write data to b
	default:
		select {
		case <-p.notifyR: // writer had wrote data
			goto start // retry write data to b
		case <-p.done:
			goto start
		}
	}
}

// Close closes the writer; subsequent reads from the
// read half of the pipe will return no bytes and EOF.
func (w *PipeWriter) Close() error {
	//notifyR(&w.r.pipe)
	return w.CloseWithError(nil)
}

// CloseWithError closes the writer; subsequent reads from the
// read half of the pipe will return no bytes and the error err,
// or EOF if err is nil.
//
// CloseWithError never overwrites the previous error if it exists.
func (w *PipeWriter) CloseWithError(err error) (e error) {
	w.r.errMux.Lock()
	e = w.r.pipe.closeWrite(err)
	w.r.errMux.Unlock()
	return
}

func (w *PipeWriter) RecycleItems() {
	w.r.RecycleItems()
}

func (p *pipe) readUnsafe(b []byte) (n int, err error) {
	bl := len(b)
start:
	if p.close.Load() {
		err = p.readCloseError()
		if //goland:noinspection GoTypeAssertionOnErrors
		_, ok := err.(*readerError); ok {
			return
		}
	}
	bPtr := unsafe.Pointer(&b[0])
	p.wr.Lock()
	if p.buf == nil {
		p.wr.Unlock()
		err = ErrBufRecycled
		return
	}
	if err != nil {
		// format error
		if p.l != 0 {
			err = nil
		} else {
			if //goland:noinspection GoDirectComparisonOfErrors
			err == WriterIoEOF {
				err = io.EOF
			}
			p.wr.Unlock()
			return
		}
	}
	if bl == 0 {
		p.wr.Unlock()
		return
	}
	// write data from buf to b
	var copyN int
	var haveL int
	space := p.c - p.l
	if p.l > 0 {
		haveL = p.w - p.r
		// copy from buf
		if haveL > 0 {
			if haveL < bl {
				copyN = haveL
			} else {
				copyN = bl
			}
			//
			p.r += copyN
			memmove(bPtr, unsafe.Add(p.bufPtr, p.r), uintptr(copyN))
		} else {
			haveL = p.c - p.r
			if haveL < bl {
				copyN = haveL
			} else {
				copyN = bl
			}
			//
			p.r += copyN
			memmove(bPtr, unsafe.Add(p.bufPtr, p.r), uintptr(copyN))
			//
			if copyN != bl {
				//
				bPtr = unsafe.Add(bPtr, copyN)
				//
				haveL = p.w
				if haveL > bl-copyN {
					copyN = bl - copyN
				} else {
					copyN = haveL
				}
				//
				memmove(bPtr, p.bufPtr, uintptr(copyN))
				p.r = copyN
			}
		}
		//
		if p.l > bl {
			p.l -= bl
			n = bl
		} else {
			n = p.l
			p.l = 0
			p.r = 0
			p.w = 0
			if //goland:noinspection GoDirectComparisonOfErrors
			err == WriterIoEOF {
				err = io.EOF
			}
		}
		p.wr.Unlock()
		if space == 0 {
			notifyW(p)
		}
		return
	}
	// no buf copy to b
	// wait sign from writer
	p.wr.Unlock()
	// nothing read wait sign from writer.
	select {
	case <-p.notifyR: // writer had wrote data
		goto start // retry write data to b
	case <-p.done:
		goto start
	}
}

func (p *pipe) read(b []byte) (n int, err error) {
	bl := len(b)
start:
	if p.close.Load() {
		err = p.readCloseError()
		if //goland:noinspection GoTypeAssertionOnErrors
		_, ok := err.(*readerError); ok {
			return
		}
	}
	p.wr.Lock()
	if p.buf == nil {
		p.wr.Unlock()
		err = ErrBufRecycled
		return
	}
	if err != nil {
		if p.l != 0 {
			err = nil
		} else {
			if //goland:noinspection GoDirectComparisonOfErrors
			err == WriterIoEOF {
				err = io.EOF
			}
			p.wr.Unlock()
			return
		}
	}
	if bl == 0 {
		p.wr.Unlock()
		return
	}
	// write data from buf to b
	space := p.c - p.l
	if p.l > 0 {
		// copy from buf
		if p.r < p.w {
			p.r += copy(b, p.buf[p.r:p.w])
		} else {
			n1 := copy(b, p.buf[p.r:])
			p.r += n1
			if n1 != bl {
				p.r = copy(b[n1:], p.buf[:p.w])
			}
		}
		//
		if p.l > bl {
			p.l -= bl
			n = bl
		} else {
			n = p.l
			p.l = 0
			//
			p.r = 0
			p.w = 0
			if //goland:noinspection GoDirectComparisonOfErrors
			err == WriterIoEOF {
				err = io.EOF
			}
		}
		p.wr.Unlock()
		if space == 0 {
			notifyW(p)
		}
		return
	}
	// no buf copy to b
	// wait sign from writer
	p.wr.Unlock()
	// nothing read wait sign from writer.
	select {
	case <-p.notifyR: // writer had wrote data
		goto start // retry write data to b
	case <-p.done:
		// write data and done may occur at the same time
		goto start
	}
}

func (p *pipe) writeUnsafe(b []byte) (n int, err error) {
	bl := len(b)
	//bPtr := unsafe.Pointer(&b[0])
	var bPtr unsafe.Pointer
start:
	if p.close.Load() {
		// when any error return
		err = p.writeCloseError()
		return
	}
	if bl == 0 {
		return
	}
	if bPtr == nil {
		bPtr = *(*unsafe.Pointer)(unsafe.Pointer(&b))
	}
	spaceL := 0
	copyN := 0
	copied := 0
	p.wr.Lock()
	if p.buf == nil {
		p.wr.Unlock()
		err = ErrBufRecycled
		return
	}
	// write b to buf
	space := p.c - p.l
	oldPL := p.l
	if space > 0 {
		if p.w < p.r {
			spaceL = p.r - p.w
			if spaceL < bl {
				copyN = spaceL
			} else {
				copyN = bl
			}
			p.w += copyN
			copied += copyN
			bl = bl - copyN
			bPtr = unsafe.Add(bPtr, copyN)
			memmove(unsafe.Add(p.bufPtr, p.w), bPtr, uintptr(copyN))
		} else {
			spaceL = p.c - p.w
			if spaceL > bl {
				copyN = bl
			} else {
				copyN = spaceL
			}
			//
			bl = bl - copyN
			bPtr = unsafe.Add(bPtr, copyN)
			memmove(unsafe.Add(p.bufPtr, p.w), bPtr, uintptr(copyN))
			p.w += copyN
			copied += copyN
			//
			if bl != 0 {
				spaceL = p.r
				if spaceL < bl {
					copyN = spaceL
				} else {
					copyN = bl
				}
				memmove(p.bufPtr, bPtr, uintptr(copyN))
				bl += copyN
				p.w = copyN
				copied += copyN
				bPtr = unsafe.Add(bPtr, copyN)
			}
		}
		if bl == 0 {
			// update p.l n
			p.l += copied
			p.wr.Unlock()
			n += copied
			if oldPL == 0 {
				notifyR(p)
			}
			// b is full copy
			return
		}
		// update p.l b n
		p.l += space
		//b = b[space:]
		n += space
	}
	p.wr.Unlock()
	//if space > 0 {
	if oldPL == 0 {
		notifyR(p)
	}
	select {
	case <-p.done:
		err = p.writeCloseError()
		return
	case <-p.notifyW:
		goto start
	}
}

func (p *pipe) write(b []byte) (n int, err error) {
start:
	if p.close.Load() {
		// when any error return
		err = p.writeCloseError()
		return
	}
	bl := len(b)
	if bl == 0 {
		return
	}
	p.wr.Lock()
	if p.buf == nil {
		p.wr.Unlock()
		err = ErrBufRecycled
		return
	}
	// write b to buf
	space := p.c - p.l
	oldPL := p.l
	if space > 0 {
		if p.w < p.r {
			p.w += copy(p.buf[p.w:p.r], b)
		} else {
			n1 := copy(p.buf[p.w:], b)
			p.w += n1
			if n1 != bl {
				p.w = copy(p.buf[:p.r], b[n1:])
			}
		}
		if space >= bl {
			// update p.l n
			p.l += bl
			p.wr.Unlock()
			n += bl
			if oldPL == 0 {
				notifyR(p)
			}
			// b is full copy
			return
		}
		// update p.l b n
		p.l += space
		b = b[space:]
		n += space
	}
	p.wr.Unlock()
	if oldPL == 0 {
		notifyR(p)
	}
	select {
	case <-p.done:
		err = p.writeCloseError()
		return
	case <-p.notifyW:
		goto start
	}
}

func (p *pipe) closeRead(err error) (e error) {
	if err == nil {
		err = ErrClosedPipe
	}
	err = &readerError{err}
	if !p.close.Load() {
		p.rerr = err
		p.close.Store(true)
		close(p.done)
		p.wr.Lock()
		if p.l > 0 {
			e = errorClosingNotEmptyPipe
		}
		p.wr.Unlock()
		return
	}
	if p.rerr != nil {
		e = errReClosingClosedPipe
		return
	}
	p.rerr = err
	return
}

func (p *pipe) closeWrite(err error) (e error) {
	if err == nil {
		err = WriterIoEOF
	} else {
		err = &writerError{err}
	}
	if !p.close.Load() {
		p.werr = err
		p.close.Store(true)
		close(p.done)
		return
	}
	if p.werr != nil {
		e = errReClosingClosedPipe
		return
	}
	p.werr = e
	return
}

// readCloseError is considered internal to the pipe type.
func (p *pipe) readCloseError() (e error) {
	p.errMux.Lock()
	e = p.rerr
	if e == nil {
		e = p.werr
	}
	p.errMux.Unlock()
	return
}

// writeCloseError is considered internal to the pipe type.
func (p *pipe) writeCloseError() (e error) {
	p.errMux.Lock()
	e = p.rerr
	if e == nil {
		e = p.werr
	}
	p.errMux.Unlock()
	return
}

func (p *pipe) flush() (err error) {
start:
	if p.close.Load() {
		// when any error return
		err = p.writeCloseError()
		return
	}
	p.wr.Lock()
	if p.l == 0 {
		p.wr.Unlock()
		return
	}
	p.wr.Unlock()
	// wait reader sign.
	select {
	case <-p.done:
		err = p.writeCloseError()
		return
	case <-p.notifyW:
		// reader had read buf; check again p.l is zero.
		goto start
	}
}

func notifyW(p *pipe) {
	select {
	case p.notifyW <- struct{}{}:
	default:
	}
}

func notifyR(p *pipe) {
	select {
	case p.notifyR <- struct{}{}:
	default:
	}
}

//go:noescape
//go:linkname memmove runtime.memmove
//goland:noinspection GoUnusedParameter
func memmove(to unsafe.Pointer, from unsafe.Pointer, n uintptr)

type readerError struct {
	err error
}
type writerError struct {
	err error
}

func (w *writerError) Error() string {
	return w.err.Error()
}
func (w *writerError) Unwrap() error {
	return w.err
}
func (r *readerError) Error() string {
	return r.err.Error()
}
func (r *readerError) Unwrap() error {
	return r.err
}

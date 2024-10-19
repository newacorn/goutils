// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package io provides basic interfaces to I/O primitives.
// Its primary job is to wrap existing implementations of such primitives,
// such as those in package os, into shared public interfaces that
// abstract the functionality, plus some other related primitives.
//
// Because these interfaces and primitives wrap lower-level operations with
// various implementations, unless otherwise informed clients should not
// assume they are safe for parallel execution.
package io

import (
	"errors"
	bpool "github.com/newacorn/simple-bytes-pool"
	"io"
	"sync"
)

// errInvalidWrite means that a write returned an impossible count.
var errInvalidWrite = errors.New("invalid write result")

// CopyN copies n bytes (or until an error) from src to dst.
// It returns the number of bytes copied and the earliest
// error encountered while copying.
// On return, written == n if and only if err == nil.
//
// If dst implements [ReaderFrom], the copy is implemented using it.
func CopyN(dst io.Writer, src io.Reader, n int64) (written int64, err error) {
	written, err = Copy(dst, io.LimitReader(src, n))
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = io.EOF
	}
	return
}

// Copy copies from src to dst until either EOF is reached
// on src or an error occurs. It returns the number of bytes
// copied and the first error encountered while copying, if any.
//
// A successful Copy returns err == nil, not err == EOF.
// Because Copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
//
// If src implements [WriterTo],
// the copy is implemented by calling src.WriteTo(dst).
// Otherwise, if dst implements [ReaderFrom],
// the copy is implemented by calling dst.ReadFrom(src).
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return copyBuffer(dst, src, nil)
}

// CopyBuffer is identical to Copy except that it stages through the
// provided buffer (if one is required) rather than allocating a
// temporary one. If buf is nil, one is allocated; otherwise if it has
// zero length, CopyBuffer panics.
//
// If either src implements [WriterTo] or dst implements [ReaderFrom],
// buf will not be used to perform the copy.
func CopyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	return copyBuffer(dst, src, buf)
}

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	var pb *bpool.Bytes
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		pb = bpool.Get(size)
		buf = pb.B
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if pb != nil {
		pb.RecycleToPool00()
	}
	return written, err
}

// CopyBufferMust is the actual implementation of Copy and CopyBuffer.
// Buf must not be nil or length 0, or panic occur.
func CopyBufferMust(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf == nil || len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// Discard is a [Writer] on which all Write calls succeed
// without doing anything.
var Discard io.Writer = discard{}
var DiscardFull = discard{}

type discard struct{}

// discard implements ReaderFrom as an optimization so Copy to
// io.Discard can avoid doing unnecessary work.
var _ io.ReaderFrom = discard{}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}

func (discard) WriteString(s string) (int, error) {
	return len(s), nil
}

var blackHolePool = sync.Pool{
	New: func() any {
		b := make([]byte, 8192)
		return &b
	},
}

func (discard) ReadFrom(r io.Reader) (n int64, err error) {
	pb := bpool.Get(8192)
	readSize := 0
	for {
		readSize, err = r.Read(pb.B)
		n += int64(readSize)
		if err != nil {
			pb.RecycleToPool00()
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}

type DiscardWithEOF interface {
	ReadFromWithEOF(r io.Reader) (n int64, err error)
}

func (discard) ReadFromWithEOF(r io.Reader) (n int64, err error) {
	pb := bpool.Get(8192)
	readSize := 0
	for {
		readSize, err = r.Read(pb.B)
		n += int64(readSize)
		if err != nil {
			pb.RecycleToPool00()
			/*
				if err == io.EOF {
					return n, nil
				}
			*/
			return
		}
	}
}

// ReadAll reads from r until an error or EOF and returns the data it read.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read from src until EOF, it does not treat an EOF from Read
// as an error to be reported.
func ReadAll(r io.Reader) (pb *bpool.Bytes, err error) {
	pb = bpool.Get(512)
	b := pb.B
	var n int
	for {
		n, err = r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			newPb := bpool.Get(len(b) + len(b)/2)
			copy(newPb.B, b)
			b = newPb.B[:len(b)]
			pb.RecycleToPool00()
			pb = newPb
		}
	}
}

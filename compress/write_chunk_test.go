package compress

import (
	"github.com/gookit/goutil/testutil/assert"
	"github.com/newacorn/cbrotli/go/cbrotli"
	"io"
	"log"
	"os"
	"testing"
)

type EmptyOut struct {
	totalLen int
}

func (e *EmptyOut) Write(p []byte) (int, error) {
	log.Println(len(p))
	e.totalLen += len(p)
	log.Println("total len", e.totalLen)
	return len(p), nil
}

// 246
func TestCompressChunkGzip(t *testing.T) {
	f, err := os.Open(curDir + "/testdata/jquery-3.7.1.js")
	defer func() { _ = f.Close() }()
	//
	assert.NoErr(t, err)
	p := DefaultGzipCompressPools.Pool(-1)
	w := p.Get()
	w.Reset(&EmptyOut{})
	allSrcBytes, _ := io.ReadAll(f)
	n, err := w.Write(allSrcBytes)
	assert.NoErr(t, err)
	t.Logf("write chunk %d", n)
	err = w.Close()
	assert.NoErr(t, err)
	p.Put(w)
}

// level 3 20KB左右每次
func TestCompressChunkCBrotliV2(t *testing.T) {
	f, err := os.Open(curDir + "/testdata/jquery-3.7.1.js")
	defer func() { _ = f.Close() }()
	//
	assert.NoErr(t, err)
	p := DefaultCBrotliCompressPools.Pool(3)
	w := p.Get()
	w.Reset(&EmptyOut{})
	allSrcBytes, _ := io.ReadAll(f)
	n, err := w.Write(allSrcBytes)
	assert.NoErr(t, err)
	t.Logf("write chunk %d", n)
	err = w.Close()
	assert.NoErr(t, err)
	p.Put(w)
}

// level 3 20KB左右每次
func TestCompressChunkCBrotli(t *testing.T) {
	f, err := os.Open(curDir + "/testdata/jquery-3.7.1.js")
	defer func() { _ = f.Close() }()
	//
	assert.NoErr(t, err)
	w := cbrotli.NewWriter(&EmptyOut{}, cbrotli.WriterOptions{Quality: 3})
	allSrcBytes, _ := io.ReadAll(f)
	n, err := w.Write(allSrcBytes)
	assert.NoErr(t, err)
	t.Logf("write chunk %d", n)
	err = w.Close()
	assert.NoErr(t, err)
}

// level 3 20KB左右每次
func TestCompressChunkBrotliV2(t *testing.T) {
	f, err := os.Open(curDir + "/testdata/jquery-3.7.1.js")
	defer func() { _ = f.Close() }()
	//
	assert.NoErr(t, err)
	p := DefaultBrotliCompressPools.Pool(3)
	w := p.Get()
	w.Reset(&EmptyOut{})
	allSrcBytes, _ := io.ReadAll(f)
	n, err := w.Write(allSrcBytes)
	assert.NoErr(t, err)
	t.Logf("write chunk %d", n)
	//
	//
	err = w.Close()
	assert.NoErr(t, err)
	p.Put(w)
}

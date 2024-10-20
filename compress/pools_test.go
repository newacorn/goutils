package compress

import (
	"github.com/gookit/goutil/testutil/assert"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/newacorn/brotli/matchfinder"
	"github.com/newacorn/cbrotli/go/cbrotli"
	"github.com/newacorn/goutils/bytes"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/xyproto/randomstring"
	"io"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

func TestLevelRangeCorrect(t *testing.T) {
	for i := -10; i < 100; i++ {
		p := DefaultGzipCompressPools.Pool(i)
		gw := p.Get().(*gzip.Writer)
		level := int(reflect.ValueOf(*gw).FieldByName("level").Int())
		//
		if i >= -2 && i <= 9 {
			assert.Eq(t, i, level)
		} else {
			assert.Eq(t, -1, level)
		}
		//
		DefaultBrotliCompressPools.Pool(i)
		//
		p2 := DefaultDeflateCompressPools.Pool(i)
		dw := p2.Get().(*zlib.Writer)
		level = int(reflect.ValueOf(*dw).FieldByName("level").Int())
		//
		if i >= -2 && i <= 9 {
			assert.Eq(t, i, level)
		} else {
			assert.Eq(t, -1, level)
		}
		//
		p3 := DefaultZstdCompressPools.Pool(i)
		zw := p3.Get().(*zstd.Encoder)
		level = int(reflect.ValueOf(*zw).FieldByName("o").FieldByName("level").Int())
		//
		if i < 5 && i > 0 {
			assert.Eq(t, i, level)
		}
		if i >= 5 && i < 12 {
			assert.Eq(t, 4, level)
		}
		if i == 0 {
			assert.Eq(t, 1, level)
		}
		if i >= 12 || i < 0 {
			assert.Eq(t, int(zstd.SpeedDefault), level)
		}
		//
		cp := DefaultCBrotliCompressPools.Pool(i)
		//
		w := cp.Get()
		if i >= 0 && i <= 11 {
			assert.Eq(t, i, w.(*cbrotli.WWriter).Quality)
		} else {
			assert.Eq(t, 3, w.(*cbrotli.WWriter).Quality)
		}
	}
}
func TestResourceLeak(t *testing.T) {
	const loopCount = 10000
	ps := []Pooler{
		DefaultGzipCompressPools.Pool(zlib.DefaultCompression),
		DefaultDeflateCompressPools.Pool(zlib.DefaultCompression),
		DefaultZstdCompressPools.Pool(int(zstd.SpeedDefault)),
		DefaultBrotliCompressPools.Pool(zlib.DefaultCompression),
		DefaultCBrotliCompressPools.Pool(cbrotli.DefaultQuality),
	}
	psNames := []string{
		"DefaultGzipCompressPools",
		"DefaultDeflateCompressPools",
		"DefaultZstdCompressPools",
		"DefaultBrotliCompressPools",
		"DefaultCBrotliCompressPools",
	}
	for i, p := range ps {
		t.Run(psNames[i], func(t *testing.T) {
			ps1, err := process.NewProcess(int32(os.Getpid()))
			assert.NoErr(t, err)
			memInfo, err := ps1.MemoryInfo()
			assert.NoErr(t, err)
			rss := memInfo.RSS
			for i := 0; i < loopCount; i++ {
				w := p.Get()
				p.Put(w)
			}
			memInfo, err = ps1.MemoryInfo()
			assert.NoErr(t, err)
			endRss := memInfo.RSS
			assert.Lt(t, endRss, rss+1<<25)
		})
	}
}
func TestAllLevelGetPut(t *testing.T) {
	const loopCount = 10000
	pcs := []interface{}{
		DefaultGzipCompressPools,
		DefaultDeflateCompressPools,
		DefaultZstdCompressPools,
		DefaultBrotliCompressPools,
		DefaultCBrotliCompressPools,
	}
	psNames := []string{
		"DefaultGzipCompressPools",
		"DefaultDeflateCompressPools",
		"DefaultZstdCompressPools",
		"DefaultBrotliCompressPools",
		"DefaultCBrotliCompressPools",
	}
	for i, pc := range pcs {
		t.Run(psNames[i], func(t *testing.T) {
			for i := 0; i <= 11; i++ {
				var p Pooler
				switch t := pc.(type) {
				case *PoolContainer[*gzip.Writer]:
					p = t.Pool(i)
				case *PoolContainer[*zlib.Writer]:
					p = t.Pool(i)
				case *PoolContainer[*zstd.Encoder]:
					p = t.Pool(i)
				case *PoolContainer[*cbrotli.WWriter]:
					p = t.Pool(i)
				case *PoolContainer[*matchfinder.Writer]:
					p = t.Pool(i)
				}
				for i := 0; i < loopCount; i++ {
					w := p.Get()
					p.Put(w)
				}
			}
		})
	}
}
func TestPoolCompressAndDecompressCBrotli(t *testing.T) {
	t.Parallel()
	const loopCount = 2
	for i := 0; i < 12; i++ {
		level := 10
		for i := 0; i < loopCount; i++ {
			p := DefaultCBrotliCompressPools.Pool(level)
			w := p.Get()
			index := rand.Int31n(int32(len(bytesContainer)))
			srcBytes := bytesContainer[index]
			buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
			w.Reset(&buf)
			_, err := w.Write(srcBytes)
			assert.NoErr(t, err)
			err = w.Close()
			r := cbrotli.NewReader(&buf)
			rs, err := io.ReadAll(r)
			assert.NoErr(t, err)
			assert.Eq(t, srcBytes, rs)
			assert.NoErr(t, err)
			buf.Reset()
			buf.RecycleItems()
			p.Put(w)
		}
	}

}
func TestPoolCompressAndDecompressGoBrotli(t *testing.T) {
	t.Parallel()
	const loopCount = 2
	for i := 0; i < 12; i++ {
		level := 10
		for i := 0; i < loopCount; i++ {
			p := DefaultBrotliCompressPools.Pool(level)
			w := p.Get()
			index := rand.Int31n(int32(len(bytesContainer)))
			srcBytes := bytesContainer[index]
			buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
			w.Reset(&buf)
			_, err := w.Write(srcBytes)
			assert.NoErr(t, err)
			err = w.Close()
			r := cbrotli.NewReader(&buf)
			rs, err := io.ReadAll(r)
			assert.NoErr(t, err)
			assert.Eq(t, srcBytes, rs)
			assert.NoErr(t, err)
			buf.Reset()
			buf.RecycleItems()
			p.Put(w)
		}
	}
}
func TestPoolCompressAndDecompressGzip(t *testing.T) {
	t.Parallel()
	const loopCount = 2
	for i := 0; i < 12; i++ {
		level := 10
		for i := 0; i < loopCount; i++ {
			p := DefaultGzipCompressPools.Pool(level)
			w := p.Get()
			index := rand.Int31n(int32(len(bytesContainer)))
			srcBytes := bytesContainer[index]
			buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
			w.Reset(&buf)
			_, err := w.Write(srcBytes)
			assert.NoErr(t, err)
			err = w.Close()
			r, err := gzip.NewReader(&buf)
			assert.NoErr(t, err)
			rs, err := io.ReadAll(r)
			assert.NoErr(t, err)
			assert.Eq(t, srcBytes, rs)
			assert.NoErr(t, err)
			buf.Reset()
			buf.RecycleItems()
			p.Put(w)
		}
	}
}

func TestPoolCompressAndDecompressDeflate(t *testing.T) {
	const loopCount = 20
	for i := 0; i < 12; i++ {
		level := 10
		for i := 0; i < loopCount; i++ {
			p := DefaultDeflateCompressPools.Pool(level)
			w := p.Get()
			index := rand.Int31n(int32(len(bytesContainer)))
			srcBytes := bytesContainer[index]
			buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
			w.Reset(&buf)
			_, err := w.Write(srcBytes)
			assert.NoErr(t, err)
			err = w.Close()
			r, err := zlib.NewReader(&buf)
			assert.NoErr(t, err)
			rs, err := io.ReadAll(r)
			assert.NoErr(t, err)
			assert.Eq(t, srcBytes, rs)
			assert.NoErr(t, err)
			buf.Reset()
			buf.RecycleItems()
			p.Put(w)
		}
	}
}

func TestPoolCompressAndDecompressZstd(t *testing.T) {
	const loopCount = 2
	for i := 0; i < 12; i++ {
		level := 10
		for i := 0; i < loopCount; i++ {
			p := DefaultZstdCompressPools.Pool(level)
			w := p.Get()
			index := rand.Int31n(int32(len(bytesContainer)))
			srcBytes := bytesContainer[index]
			buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
			w.Reset(&buf)
			_, err := w.Write(srcBytes)
			assert.NoErr(t, err)
			err = w.Close()
			r, err := zstd.NewReader(&buf)
			assert.NoErr(t, err)
			rs, err := io.ReadAll(r)
			assert.NoErr(t, err)
			assert.Eq(t, srcBytes, rs)
			assert.NoErr(t, err)
			buf.Reset()
			buf.RecycleItems()
			p.Put(w)
		}
	}
}
func TestReaderPoolGzip(t *testing.T) {
	const dataLen = 1200
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	w := DefaultGzipCompressPools.Pool(-1).Get()
	buf := bytes.NewBufferWithSize(dataLen)
	w.Reset(buf)
	_, err := w.Write(dataBytes)
	assert.NoErr(t, err)
	//
	err = w.Close()
	r := DefaultGzipReaderPool.Get()
	err = r.Reset(buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, dataBytes, rs)
	err = r.Close()
	assert.NoErr(t, err)
	buf.Reset()
	buf.RecycleItems()
	DefaultGzipCompressPools.Pool(-1).Put(w)
	DefaultGzipReaderPool.Put(r)
}
func TestReaderPoolDeflate(t *testing.T) {
	const dataLen = 1200
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	w := DefaultDeflateCompressPools.Pool(-1).Get()
	buf := bytes.NewBufferWithSize(dataLen)
	w.Reset(buf)
	_, err := w.Write(dataBytes)
	assert.NoErr(t, err)
	//
	err = w.Close()
	r := DefaultDeflateReaderPool.Get()
	err = r.Reset(buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, dataBytes, rs)
	err = r.Close()
	assert.NoErr(t, err)
	buf.Reset()
	buf.RecycleItems()
	DefaultDeflateCompressPools.Pool(-1).Put(w)
	DefaultDeflateReaderPool.Put(r)
}
func TestReaderPoolBrotli(t *testing.T) {
	const dataLen = 1200
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	w := DefaultBrotliCompressPools.Pool(3).Get()
	buf := bytes.NewBufferWithSize(dataLen)
	w.Reset(buf)
	_, err := w.Write(dataBytes)
	assert.NoErr(t, err)
	//
	err = w.Close()
	r := DefaultBrotliReaderPool.Get()
	err = r.Reset(buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, dataBytes, rs)
	err = r.Close()
	assert.NoErr(t, err)
	buf.Reset()
	buf.RecycleItems()
	DefaultBrotliCompressPools.Pool(3).Put(w)
	DefaultBrotliReaderPool.Put(r)
}

func TestReaderPoolCBrotli(t *testing.T) {
	const dataLen = 1200
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	w := DefaultCBrotliCompressPools.Pool(3).Get()
	buf := bytes.NewBufferWithSize(dataLen)
	w.Reset(buf)
	_, err := w.Write(dataBytes)
	assert.NoErr(t, err)
	//
	err = w.Close()
	r := DefaultCBrotliReaderPool.Get()
	err = r.Reset(buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, dataBytes, rs)
	err = r.Close()
	assert.NoErr(t, err)
	buf.Reset()
	buf.RecycleItems()
	DefaultCBrotliCompressPools.Pool(3).Put(w)
	DefaultCBrotliReaderPool.Put(r)
}
func TestReaderPoolZstd(t *testing.T) {
	const dataLen = 1200
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	w := DefaultZstdCompressPools.Pool(3).Get()
	buf := bytes.NewBufferWithSize(dataLen)
	w.Reset(buf)
	_, err := w.Write(dataBytes)
	assert.NoErr(t, err)
	//
	err = w.Close()
	r := DefaultZstdReaderPool.Get()
	err = r.Reset(buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, dataBytes, rs)
	//r.Close()
	r.NilSrc()
	assert.NoErr(t, err)
	buf.Reset()
	buf.RecycleItems()
	DefaultZstdCompressPools.Pool(3).Put(w)
	DefaultZstdReaderPool.Put(r)
}
func TestZstdReader(t *testing.T) {
	po := DefaultZstdCompressPools.Pool(int(zstd.SpeedDefault))
	w := po.Get()
	buf := bytes.Buffer{}
	w.Reset(&buf)
	_, err := w.Write([]byte("hello world"))
	assert.NoErr(t, err)
	err = w.Close()
	assert.NoErr(t, err)
	r, err := zstd.NewReader(&buf)
	assert.NoErr(t, err)
	rs, err := io.ReadAll(r)
	assert.NoErr(t, err)
	assert.Eq(t, []byte("hello world"), rs)
	po.Put(w)
	// can not call close. close prevent reader resuse.
	r.NilSrc()
	err = r.Reset(&buf)
	assert.NoErr(t, err)
}

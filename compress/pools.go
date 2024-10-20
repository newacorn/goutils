package compress

import (
	"bytes"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/newacorn/brotli"
	"github.com/newacorn/brotli/matchfinder"
	"github.com/newacorn/cbrotli/go/cbrotli"
	"sync/atomic"

	"io"
	"runtime"
	"strings"
	"sync"
)

//goland:noinspection GoNameStartsWithPackageName
type CompressInfo struct {
	level int
	pw    io.WriteCloser
	src   io.Reader
}

var _ Writer = (*matchfinder.Writer)(nil)

//goland:noinspection GoNameStartsWithPackageName
type CompressInfoChan chan *CompressInfo

var InitCount atomic.Int64

const levelCount = 12

var DefaultGzipCompressPools = &PoolContainer[*gzip.Writer]{
	//poolChan:     gzipCompressChan,
	defaultLevel: gzip.DefaultCompression,
	offset:       2,
	poolsInit: func() [12]*CompressPool[*gzip.Writer] {
		var pools [12]*CompressPool[*gzip.Writer]
		for i := range pools {
			level := i - 2
			pools[i] = &CompressPool[*gzip.Writer]{Pool: sync.Pool{
				New: func() any {
					InitCount.Add(1)
					w, err := gzip.NewWriterLevel(nil, level)
					_ = err
					return w
				}}, needBuffer: true}
		}
		return pools
	},
}

var DefaultZstdCompressPools = &PoolContainer[*zstd.Encoder]{
	defaultLevel: int(zstd.SpeedDefault),
	offset:       0,
	poolsInit: func() [12]*CompressPool[*zstd.Encoder] {
		var pools [12]*CompressPool[*zstd.Encoder]
		for i := range pools {
			level := i
			if level >= 5 {
				level = int(zstd.SpeedBestCompression)
			}
			if level <= 0 {
				level = int(zstd.SpeedFastest)
			}
			pools[i] = &CompressPool[*zstd.Encoder]{Pool: sync.Pool{
				New: func() any {
					w, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
					_ = err
					return w
				}}}
		}
		return pools
	},
}

var DefaultDeflateCompressPools = &PoolContainer[*zlib.Writer]{
	defaultLevel: zlib.DefaultCompression,
	offset:       2,
	poolsInit: func() [12]*CompressPool[*zlib.Writer] {
		var pools [12]*CompressPool[*zlib.Writer]
		for i := range pools {
			level := i - 2
			pools[i] = &CompressPool[*zlib.Writer]{Pool: sync.Pool{
				New: func() any {
					w, _ := zlib.NewWriterLevel(nil, level)
					return w
				}}, needBuffer: true}
		}
		return pools
	},
}
var DefaultBrotliCompressPools = &PoolContainer[*matchfinder.Writer]{
	defaultLevel: 4,
	offset:       0,
	poolsInit: func() [12]*CompressPool[*matchfinder.Writer] {
		var pools [12]*CompressPool[*matchfinder.Writer]
		for i := range pools {
			level := i
			pools[i] = &CompressPool[*matchfinder.Writer]{Pool: sync.Pool{
				New: func() any {
					w := brotli.NewWriterV2(nil, level)
					return w
				}}}
		}
		return pools
	},
}
var DefaultCBrotliCompressPools = &PoolContainer[*cbrotli.WWriter]{
	defaultLevel: 3,
	offset:       0,
	poolsInit: func() [12]*CompressPool[*cbrotli.WWriter] {
		var pools [12]*CompressPool[*cbrotli.WWriter]
		for i := range pools {
			level := i
			pools[i] = &CompressPool[*cbrotli.WWriter]{Pool: sync.Pool{
				New: func() any {
					w := cbrotli.NewWWriter(nil, cbrotli.WriterV2Options{
						Quality: level,
					})
					runtime.SetFinalizer(w, func(w *cbrotli.WWriter) {
						w.Destroy()
					})
					return w
				}}}
		}
		return pools
	},
}

//goland:noinspection GoNameStartsWithPackageName
type CompressCBrotliPools struct {
	defaultLevel       int
	minLevel, maxLevel int
}

func init() {
	DefaultZstdCompressPools.pools = DefaultZstdCompressPools.poolsInit()
	DefaultGzipCompressPools.pools = DefaultGzipCompressPools.poolsInit()
	DefaultBrotliCompressPools.pools = DefaultBrotliCompressPools.poolsInit()
	DefaultDeflateCompressPools.pools = DefaultDeflateCompressPools.poolsInit()
	DefaultCBrotliCompressPools.pools = DefaultCBrotliCompressPools.poolsInit()
}

type comWriter interface {
	*gzip.Writer | *zlib.Writer | *matchfinder.Writer | *zstd.Encoder | *cbrotli.WWriter
}

//goland:noinspection GoNameStartsWithPackageName
type CompressPool[T comWriter] struct {
	sync.Pool
	needBuffer bool
}

func (c *CompressPool[T]) NeedBuffer() bool {
	return c.needBuffer
}
func (c *CompressPool[T]) Get() Writer {
	var t T
	_ = Writer(t)
	return c.Pool.Get().(Writer)
}
func (c *CompressPool[T]) Put(compressW Writer) {
	var t T
	_ = Writer(t)
	c.Pool.Put(compressW)
}

type PoolContainerInter interface {
	Pool(level int) Pooler
}

//goland:noinspection GoNameStartsWithPackageName
type PoolContainer[T comWriter] struct {
	pools        [levelCount]*CompressPool[T]
	defaultLevel int
	offset       int
	poolsInit    func() [levelCount]*CompressPool[T]
}

//goland:noinspection GoNameStartsWithPackageName
type Pooler interface {
	Get() Writer
	Put(Writer)
	NeedBuffer() bool
}

func (cps *PoolContainer[T]) Pool(level int) *CompressPool[T] {
	level = cps.offset + level
	if level < 0 || level >= levelCount {
		level = cps.defaultLevel + cps.offset
	}
	if uint(level) < uint(len(cps.pools)) {
		// bounding check ease
		return cps.pools[uint(level)]
	}
	panic("never happen")
}

type ReadResetCloser interface {
	io.ReadCloser
	Reset(r io.Reader) error
}
type DeflateReaderInter interface {
	io.ReadCloser
	zlib.Resetter
}

var DefaultGzipReaderPool ReaderPool[*gzip.Reader]
var DefaultDeflateReaderPool ReaderPool[DeflateReader]
var DefaultDeflateReaderDictPool = &DefaultDeflateReaderPool.Pool
var DefaultBrotliReaderPool ReaderPool[*brotli.Reader]
var DefaultCBrotliReaderPool CBrotliReaderPool
var DefaultZstdReaderPool ReaderPool[ZstdReader]

type CBrotliReaderPool struct{}
type DeflateReaderPool struct{ sync.Pool }

func (b CBrotliReaderPool) Get() *cbrotli.ReaderV2 {
	return cbrotli.NewReaderV2(nil)
}

func (b CBrotliReaderPool) Put(r *cbrotli.ReaderV2) {
	_ = r.Close()
}

type comReader interface {
	*gzip.Reader | *brotli.Reader | ZstdReader | DeflateReader
}
type ReaderPool[T comReader] struct {
	sync.Pool
}

func (b *ReaderPool[T]) Get() T {
	return b.Pool.Get().(T)
}
func (b *ReaderPool[T]) Put(w T) {
	b.Pool.Put(w)
}

type DeflateReader struct {
	DeflateReaderInter
}

func (d DeflateReader) Reset(r io.Reader) (err error) {
	return d.DeflateReaderInter.Reset(r, nil)
}

func (d DeflateReader) ResetDict(r io.Reader, dict []byte) (err error) {
	return d.DeflateReaderInter.Reset(r, dict)
}

type ZstdReader struct {
	*zstd.Decoder
}

func (z ZstdReader) Close() error {
	z.Decoder.NilSrc()
	return nil
}

var (
	tmpGzipStr    string
	tmpDeflateStr string
)

func init() {
	buf := bytes.Buffer{}
	tmpW, _ := gzip.NewWriterLevel(&buf, 1)
	_, _ = tmpW.Write([]byte(""))
	_ = tmpW.Close()
	tmpGzipStr = buf.String()
	buf.Reset()
	tmpW2, _ := zlib.NewWriterLevel(&buf, 1)
	_, _ = tmpW2.Write([]byte(""))
	_ = tmpW2.Close()
	tmpDeflateStr = buf.String()
}
func init() {

	DefaultBrotliReaderPool.New = func() interface{} {
		return brotli.NewReader(nil)
	}

	DefaultGzipReaderPool.New = func() interface{} {
		tmpR := strings.NewReader(tmpGzipStr)
		r, _ := gzip.NewReader(tmpR)
		return r
	}

	DefaultDeflateReaderPool.New = func() interface{} {
		tmpR := strings.NewReader(tmpDeflateStr)
		r, _ := zlib.NewReader(tmpR)
		return DeflateReader{DeflateReaderInter: r.(DeflateReaderInter)}
	}

	DefaultZstdReaderPool.New = func() interface{} {
		r, _ := zstd.NewReader(nil)
		return ZstdReader{Decoder: r}
	}
}

type Writer interface {
	Write([]byte) (int, error)
	Close() error
	Flush() error
	Reset(io.Writer)
}

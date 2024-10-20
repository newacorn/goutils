package compress

import (
	"bytes"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/newacorn/brotli"
	"slices"
	"unsafe"
)

const (
	DeflateNoCompression       deflateLevel = flate.NoCompression
	DeflateBestSpeed                        = flate.BestSpeed
	DeflateBestCompression                  = flate.BestCompression
	DeflateDefaultCompression               = flate.DefaultCompression
	DeflateConstantCompression              = flate.ConstantCompression
	DeflateHuffmanOnly                      = flate.HuffmanOnly
)
const (
	GzipNoCompression       gzipLevel = zlib.NoCompression
	GzipBestSpeed                     = zlib.BestSpeed
	GzipBestCompression               = zlib.BestCompression
	GzipDefaultCompression            = zlib.DefaultCompression
	GzipConstantCompression           = zlib.ConstantCompression
	GzipHuffmanOnly                   = zlib.HuffmanOnly
)

const (
	ZstdSpeedNotSetEncoderLevel zstdLevel = iota
	ZstdSpeedFastest
	ZstdSpeedDefault
	ZstdSpeedBetterCompression
	ZstdSpeedBestCompression
	ZstdSpeedLast
)

const (
	BrotliBestSpeed          brotliLevel = brotli.BestSpeed
	BrotliBestCompression    brotliLevel = brotli.BestCompression
	BrotliDefaultCompression brotliLevel = 4
	BrotliHighCompression    brotliLevel = 6
)

type zstdLevel zstd.EncoderLevel
type gzipLevel int
type brotliLevel int
type deflateLevel int

type Levels struct {
	GzipLevel    gzipLevel
	ZstdLevel    zstdLevel
	BrotliLevel  brotliLevel
	DeflateLevel deflateLevel
}
type Level int

const (
	GzipDefaultLevel    Level = -1
	DeflateDefaultLevel       = -1
	ZstdDefaultLevel          = Level(zstd.SpeedDefault)
	BrotliDefaultLevel        = 3
)

type Order string

const (
	Gzip    Order = "gzip"
	Deflate       = "deflate"
	Zstd          = "zstd"
	Br            = "br"
	Dump          = "dump"
)

type Info struct {
	Level int
	Pool  Pooler
}

func Pool(level int, order Order) Pooler {
	switch order {
	case Gzip:
		return DefaultGzipCompressPools.Pool(level)
	case Deflate:
		return DefaultDeflateCompressPools.Pool(level) // Deflate is default
	case Zstd:
		return DefaultZstdCompressPools.Pool(level)
	case Br:
		return DefaultCBrotliCompressPools.Pool(level)
	}
	return nil
}

var compressedMimes = [][]byte{
	[]byte("application/javascript"),
	[]byte("application/json"),
	[]byte("application/xml"),
	[]byte("image/svg+xml"),
	//
	[]byte("text/html"),
	[]byte("text/css"),
	[]byte("text/plain"),
	[]byte("text/xml"),
	//
	[]byte("text/csv"),
	[]byte("text/yaml"),
	[]byte("text/markdown"),
}

var applicationPrefixHash uint32
var textPrefixHash uint32
var imagePrefixHash uint32
var applicationMimes []uint32
var textMimes []uint32
var imageMimes [1]uint32

const applicationPrefixL = len("application")
const imagePrefixL = len("image")
const texPrefixL = len("text")

func init() {
	a := [4]byte{'a', 'p', 'p', 'l'}
	applicationPrefixHash = *(*uint32)(unsafe.Pointer(&a[0]))
	t := [4]byte{'t', 'e', 'x', 't'}
	textPrefixHash = *(*uint32)(unsafe.Pointer(&t[0]))
	i := [4]byte{'i', 'm', 'a', 'g'}
	imagePrefixHash = *(*uint32)(unsafe.Pointer(&i[0]))
	//
	for _, m := range compressedMimes {
		if bytes.HasPrefix(m, []byte("application/")) {
			applicationMimes = append(applicationMimes, *(*uint32)(unsafe.Pointer(&m[applicationPrefixL])))
		}
		if bytes.HasPrefix(m, []byte("text/")) {
			textMimes = append(textMimes, *(*uint32)(unsafe.Pointer(&m[texPrefixL])))
		}
		if bytes.HasPrefix(m, []byte("image/")) {
			imageMimes[0] = *(*uint32)(unsafe.Pointer(&m[imagePrefixL]))
		}
	}
}

const applicationMinLen = len("application/xml")
const textMinLen = len("text/csv")
const imageMinLen = len("image/svg+xml")

func CheckMimeOk(mime []byte) (ok bool) {
	l := len(mime)
	if l < 4 {
		return
	}
	//
	m := *(*uint32)(unsafe.Pointer(&mime[0]))
	if m == applicationPrefixHash {
		if l < applicationMinLen {
			return
		}
		m1 := *(*uint32)(unsafe.Pointer(&mime[applicationPrefixL]))
		if slices.Index(applicationMimes, m1) != -1 {
			ok = true
		}
		return
	}
	//
	if m == textPrefixHash {
		if l < textMinLen {
			return
		}
		m1 := *(*uint32)(unsafe.Pointer(&mime[texPrefixL]))
		if slices.Index(textMimes, m1) != -1 {
			ok = true
		}
		return
	}
	if m == imagePrefixHash {
		if l < imageMinLen {
			return
		}
		m1 := *(*uint32)(unsafe.Pointer(&mime[imagePrefixL]))
		ok = imageMimes[0] == m1
	}
	return
}

package compress

import (
	"encoding/json"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/newacorn/brotli"
	"github.com/newacorn/cbrotli/go/cbrotli"
	"github.com/newacorn/goutils/bytes"
	"github.com/xyproto/randomstring"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

type Duration int64

func (d Duration) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("%s", time.Duration(d))
	return json.Marshal(s)
}

var filePath string
var curDir string

func init() {
	var err error
	curDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkCompressRecommendLevel(b *testing.B) {
	for _, bs := range bytesContainer {

		bs := bs
		b.Logf("compressed data len :%d\n", len(bs))
		b.ReportAllocs()
		b.ResetTimer()
		b.Run("GoBrotliV2 2", func(b *testing.B) {
			var compressedLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultBrotliCompressPools.Pool(4)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("GoBrotliV2 2 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressedLen)/float64(len(bs)))
		})
		b.Run("Gzip -1", func(b *testing.B) {
			var compressedLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultGzipCompressPools.Pool(-1)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("Gzip -1 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressedLen)/float64(len(bs)))
		})
		b.Run("CBrotli-NoPool 3", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 3})
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 3 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
		b.Run("CBrotli 3", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultCBrotliCompressPools.Pool(3)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli 3 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
		b.Run("CBrotli-NoPool 4", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w := cbrotli.NewWWriter(&buf, cbrotli.WriterV2Options{Quality: 4})
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 4 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
		b.Run("CBrotli 4", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultCBrotliCompressPools.Pool(4)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli 4 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
		b.Run("CBrotli-NoPool 5", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli-NoPool 5 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
		b.Run("CBrotli 5", func(b *testing.B) {
			var compressLen int
			buf := bytes.NewBufferSizeNoPtr(len(bs))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultCBrotliCompressPools.Pool(5)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, bytes.NewBuffer(bs))
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli 5 compress data len:%d, ratio:%.3f\n", len(bs), float64(compressLen)/float64(len(bs)))
		})
	}
}

func BenchmarkVueJsFileCompress(b *testing.B) {
	benchMarkCompressFile(b, filePath)
}
func BenchmarkJqueryFileCompress(b *testing.B) {
	benchMarkCompressFile(b, curDir+"/testdata/jquery-3.7.1.js")
}

func benchMarkCompressFile(b *testing.B, filepath string) {
	f, err := os.Open(filepath)
	defer f.Close()
	if err != nil {
		b.Fatal(err)
	}
	srcBytes, err := io.ReadAll(f)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("origin size:%d\n", len(srcBytes))
	b.Run("Gzip -1 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultGzipCompressPools.Pool(-1)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Gzip -1 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("Gzip 6 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultGzipCompressPools.Pool(6)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Gzip 6 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})

	b.Run("Gzip 7 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultGzipCompressPools.Pool(7)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Gzip 7 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("GoBrotliV2 1 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(1)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotliV2 1 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("GoBrotliV2 2 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(2)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotliV2 2 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("GoBrotliV2 3 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(3)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotliV2 3 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("GoBrotliV2 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(4)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotli 4 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("GoBrotliV2 5 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(5)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotli 5 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("GoBrotliV2 6 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(6)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotli 6 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("GoBrotliV2 7 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultBrotliCompressPools.Pool(7)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("GoBrotli 7 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("CBrotli-Pool 2 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(2)
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 2})
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 2 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("CBrotli 2 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 2})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 2 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 2 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 2})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 2 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 3 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(3)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 3 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("CBrotli 3 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 3})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 3 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 3 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 3})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 3 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})

	b.Run("CBrotli-Pool 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 4})
			p := DefaultCBrotliCompressPools.Pool(4)
			w := p.Get()
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 4})
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 4  compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 4})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 4 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotliV2 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 4})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotliV2 4 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 5 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(5)
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 5})
			w := p.Get()
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 5})
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 5 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 5 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 5 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotliV2 5 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 5})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotliV2 5 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 6 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(6)
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 6})
			w := p.Get()
			//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 6})
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 6 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 6 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 6})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 6 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotliV2 6 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 6})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotliV2 6 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 7 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(7)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 7 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})

	b.Run("CBrotli 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 4})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 4 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotliV2 7 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriterV2(&buf, cbrotli.WriterV2Options{Quality: 7})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotliV2 7 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli 8 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 8})
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
		}
		b.Logf("CBrotli 8 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 9 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(9)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 9 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 10 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(10)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 10 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("CBrotli-Pool 11 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultCBrotliCompressPools.Pool(11)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("CBrotli-Pool 11 compressed ratio:%.3f", float64(compressLen)/float64(len(srcBytes)))
	})
	b.Run("Zstd 1 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultZstdCompressPools.Pool(1)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Zstd 1 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("Zstd 2 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultZstdCompressPools.Pool(2)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Zstd 2 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("Zstd 3 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultZstdCompressPools.Pool(3)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Zstd 3 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
	b.Run("Zstd 4 compressed ratio", func(b *testing.B) {
		buf := bytes.NewBufferSizeNoPtr(len(srcBytes))
		var compressLen int
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := DefaultZstdCompressPools.Pool(4)
			w := p.Get()
			w.Reset(&buf)
			_, err = io.Copy(w, bytes.NewBuffer(srcBytes))
			if err != nil {
				b.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				b.Fatal(err)
			}
			compressLen = buf.Len()
			buf.Reset()
			p.Put(w)
		}
		b.Logf("Zstd 4 compressed ratio:%.3f; %d", float64(compressLen)/float64(len(srcBytes)), compressLen)
	})
}

var bytesContainer [][]byte

func init() {
	for i := 10; i <= 20; i++ {
		dataStr := randomstring.HumanFriendlyString(2 << i)
		bytesContainer = append(bytesContainer, []byte(dataStr))
	}
}

type CompressRatioInfo struct {
	Name    string
	DataLen int
	Ratio   float64
}

var brotliV1Pool = sync.Pool{
	New: func() interface{} {
		w := brotli.NewWriterLevel(nil, 3)
		return w
	},
}

func BenchmarkCompressGzipBrotliZstd(b *testing.B) {
	for _, bytes_ := range bytesContainer {
		bytes2 := bytes_
		b.Run(fmt.Sprintf("Gzip -1 compress data len:%d", len(bytes2)), func(b *testing.B) {
			var compressedLen int
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultGzipCompressPools.Pool(-1)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				p.Put(w)
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("Gzip -1 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("Gzip 6 compress data len:%d", len(bytes2)), func(b *testing.B) {
			var compressedLen int
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultGzipCompressPools.Pool(6)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				p.Put(w)
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("Gzip 6 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("Gzip 7 compress data len:%d", len(bytes2)), func(b *testing.B) {
			var compressedLen int
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultGzipCompressPools.Pool(7)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				p.Put(w)
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("Gzip 7 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 2 compress data len:%d", len(bytes2)), func(b *testing.B) {
			//buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 2})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 2 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 3 compress data len:%d", len(bytes2)), func(b *testing.B) {
			//buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 3})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 3 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 4 compress data len:%d", len(bytes2)), func(b *testing.B) {
			//buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 4})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 4 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 5 compress data len:%d", len(bytes2)), func(b *testing.B) {
			//buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 5 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("BrotiliV2 2 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				//p := DefaultBrotliCompressPools.Pool(2)
				w := brotli.NewWriterLevel(&buf, 2)
				srcBuf := bytes.NewBuffer(bytes2)
				//w := p.Get()
				//w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				//p.Put(w)
				buf.Reset()
			}
			b.Logf("BrotiliV2 2 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))

		})
		b.Run(fmt.Sprintf("BrotiliV2 3 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultBrotliCompressPools.Pool(3)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				p.Put(w)
				buf.Reset()
			}
			b.Logf("BrotiliV2 3 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))

		})
		b.Run(fmt.Sprintf("BrotiliV2 4 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultBrotliCompressPools.Pool(4)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				p.Put(w)
				buf.Reset()
			}
			b.Logf("BrotiliV2 4 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))

		})
		b.Run(fmt.Sprintf("BrotiliV2 5 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := DefaultBrotliCompressPools.Pool(5)
				srcBuf := bytes.NewBuffer(bytes2)
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				p.Put(w)
				buf.Reset()
			}
			b.Logf("BrotiliV2 5 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))

		})
		b.Run(fmt.Sprintf("BrotiliV1 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := brotliV1Pool.Get().(*brotli.Writer)
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				brotliV1Pool.Put(w)
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("BrotiliV1 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})

		b.Run(fmt.Sprintf("zstd compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.NewBufferSizeNoPtr(len(bytes2))
			var compressLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				p := DefaultZstdCompressPools.Pool(int(zstd.SpeedDefault))
				w := p.Get()
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				brotliV1Pool.Put(w)
				compressLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("zstd compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressLen)/float64(len(bytes2)))

		})
	}
}

func BenchmarkCompressGzipBrotliZstd3(b *testing.B) {
	for _, bytes_ := range bytesContainer {
		bytes2 := bytes_
		b.Run(fmt.Sprintf("CBrotli 2 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 2})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 2 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli-pool 2 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				p := DefaultCBrotliCompressPools.Pool(2)
				w := p.Get()
				w.Reset(&buf)
				//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 2})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli-pool 2 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 3 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 3})
				//
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 3 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli-pool 3 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				p := DefaultCBrotliCompressPools.Pool(3)
				w := p.Get()
				//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 3})
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli-pool 3 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 4 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 4})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 4 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli-pool 4 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				p := DefaultCBrotliCompressPools.Pool(4)
				w := p.Get()
				//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 4})
				w.Reset(&buf)
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli-pool 4 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli 5 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				w := cbrotli.NewWriter(&buf, cbrotli.WriterOptions{Quality: 5})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
			}
			b.Logf("CBrotli 5 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
		b.Run(fmt.Sprintf("CBrotli-pool 5 compress data len:%d", len(bytes2)), func(b *testing.B) {
			buf := bytes.Buffer{}
			var compressedLen int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				srcBuf := bytes.NewBuffer(bytes2)
				p := DefaultCBrotliCompressPools.Pool(5)
				w := p.Get()
				w.Reset(&buf)
				//w := cbrotli.NewWriter(&buf, cbrotli.WriterV2Options{Quality: 5})
				_, err := io.Copy(w, srcBuf)
				if err != nil {
					b.Fatal(err)
				}
				err = w.Close()
				if err != nil {
					b.Fatal(err)
				}
				compressedLen = buf.Len()
				buf.Reset()
				p.Put(w)
			}
			b.Logf("CBrotli-pool 5 compress data len:%d, ratio:%.3f\n", len(bytes2), float64(compressedLen)/float64(len(bytes2)))
		})
	}
}

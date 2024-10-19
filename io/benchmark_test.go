package io

import (
	"github.com/gookit/goutil/testutil/assert"
	"github.com/xyproto/randomstring"
	"log"
	"sync"
	"testing"
)

func TestPipeWrite(t *testing.T) {
	dataLen := 1500
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	//
	pw, pr := Pipe()
	total := 0
	n, err := pw.write(dataBytes)
	total += n
	assert.NoErr(t, err)
	n, err = pw.write(dataBytes)
	total += n
	assert.NoErr(t, err)
	log.Println(total)
	_ = pr
}

func BenchmarkPipe(b *testing.B) {
	const dataLen = 1500
	pr, pw := Pipe()
	dataStr := randomstring.HumanFriendlyString(1500)

	dataBytes := []byte(dataStr)

	b.ResetTimer()
	b.ReportAllocs()
	b.Run("pipe", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		// reader
		rcount := 0
		wcount := 0
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			dataBytes2 := []byte(dataStr)
			for i := 0; i < n; i++ {
				subTotal := 0
				for {
					n1, _ := pr.Read(dataBytes2[subTotal:])
					subTotal += n1
					if subTotal == dataLen {
						break
					}
				}
				rcount += subTotal
			}
		}(b.N)
		// writer
		go func(n int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				n, _ := pw.Write(dataBytes)
				wcount += n
			}
		}(b.N)
		wg.Wait()
		assert.Eq(b, rcount, wcount)
	})

	b.Run("pipeUnsafe", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		// reader
		rcount := 0
		wcount := 0
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			dataBytes2 := []byte(dataStr)
			for i := 0; i < n; i++ {
				subTotal := 0
				for {
					n1, _ := pr.ReadUnsafe(dataBytes2[subTotal:])
					subTotal += n1
					if subTotal == dataLen {
						break
					}
				}
				rcount += subTotal
			}
		}(b.N)
		// writer
		go func(n int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				n, _ := pw.WriteUnsafe(dataBytes)
				wcount += n
			}
		}(b.N)
		wg.Wait()
		assert.Eq(b, rcount, wcount)
	})
}

package io

import (
	"github.com/gookit/goutil/testutil/assert"
	"github.com/newacorn/goutils/bytes"
	"github.com/xyproto/randomstring"
	"io"
	"testing"
	"time"
)

func TestWriteTo(t *testing.T) {
	const dataLen = 1024
	dataStr := randomstring.HumanFriendlyString(dataLen)
	dataBytes := []byte(dataStr)
	pr, pw := Pipe()
	defer func(pr *PipeReader) {
		err := pr.Close()
		assert.NoErr(t, err)
	}(pr)
	go func() {
		total := 0
		for i := 0; i < 10; i++ {
			n1, err1 := pw.Write(dataBytes[i*(dataLen/10) : i*(dataLen/10)+(dataLen/10)])
			total += n1
			assert.NoErr(t, err1)
			time.Sleep(time.Millisecond * 100)
		}
		n, err := pw.Write(dataBytes[(dataLen/10)*10:])
		assert.NoErr(t, err)
		total += n
		assert.Eq(t, dataLen, total)
		time.Sleep(time.Millisecond * 100)
		err = pw.CloseWithError(nil)
		assert.NoErr(t, err)
	}()
	buf := bytes.NewBufferSizeNoPtr(dataLen)
	defer buf.RecycleItems()
	total := 0
	for {
		n, err := io.Copy(&buf, pr)
		total += int(n)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
	assert.Eq(t, dataLen, total)
	assert.Eq(t, dataBytes, buf.Bytes())
	time.Sleep(time.Second * 1)
}

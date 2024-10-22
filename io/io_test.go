package io

import (
	"github.com/gookit/goutil/testutil/assert"
	"io"
	"os"
	"testing"
)

func TestReadAll(t *testing.T) {
	F, err := os.Open("./io_test.go")
	defer func() {
		_ = F.Close()
	}()
	assert.Nil(t, err)
	p, err := ReadAll(F)
	assert.Nil(t, err)
	_, _ = F.Seek(0, io.SeekStart)
	b, err := io.ReadAll(F)
	assert.Nil(t, err)
	assert.Eq(t, p.B, b)
	p.Release()
}

package objectstore

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingReaderAt struct {
	r     *bytes.Reader
	calls int
}

func (c *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	c.calls++
	return c.r.ReadAt(p, off)
}

func TestWindowedReducesUnderlyingReads(t *testing.T) {
	data := make([]byte, 10_000)
	for i := range data {
		data[i] = byte(i)
	}
	under := &countingReaderAt{r: bytes.NewReader(data)}
	w := newWindowed(under, int64(len(data)), 4096)

	got := make([]byte, 0, len(data))
	buf := make([]byte, 100)
	for off := int64(0); off < w.Size(); off += 100 {
		n, err := w.ReadAt(buf, off)
		got = append(got, buf[:n]...)
		if err != nil {
			require.ErrorIs(t, err, io.EOF)
		}
	}
	assert.Equal(t, data, got)
	assert.Equal(t, 3, under.calls, "100 leituras pequenas viram 3 requisições")
}

func TestWindowedReadAcrossWindowAndEOF(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefgh"), 1000)
	w := newWindowed(bytes.NewReader(data), int64(len(data)), 1000)
	p := make([]byte, 3000)
	n, err := w.ReadAt(p, 7000)
	assert.Equal(t, 1000, n)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, data[7000:], p[:n])
}

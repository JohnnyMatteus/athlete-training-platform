// Package objectstore lê arquivos de importação do MinIO/S3.
package objectstore

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
)

// Store implementa port.ObjectStore.
type Store struct{ client *minio.Client }

var _ port.ObjectStore = (*Store)(nil)

func New(client *minio.Client) *Store { return &Store{client: client} }

type object struct {
	*windowed
	closer io.Closer
}

func (o object) Close() error { return o.closer.Close() }

func (s *Store) Open(ctx context.Context, bucket, key string) (port.Object, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s/%s: %w", bucket, key, err)
	}
	st, err := obj.Stat() // falha aqui se o objeto não existe
	if err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("stat %s/%s: %w", bucket, key, err)
	}
	return object{windowed: newWindowed(obj, st.Size, DefaultWindow), closer: obj}, nil
}

// DefaultWindow é o tamanho da janela de leitura antecipada.
const DefaultWindow = 4 << 20

// windowed evita um GET por leitura pequena: archive/zip e o decoder de flate leem em
// pedaços de poucos KB, e cada ReadAt direto viraria uma requisição ao S3.
// Não é seguro para uso concorrente; o worker lê o arquivo sequencialmente.
type windowed struct {
	ra   io.ReaderAt
	size int64
	win  int
	buf  []byte
	off  int64 // posição do início de buf
}

func newWindowed(ra io.ReaderAt, size int64, win int) *windowed {
	return &windowed{ra: ra, size: size, win: win, off: -1}
}

func (w *windowed) Size() int64 { return w.size }

func (w *windowed) ReadAt(p []byte, off int64) (int, error) {
	if off >= w.size {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && off < w.size {
		if w.off < 0 || off < w.off || off >= w.off+int64(len(w.buf)) {
			if err := w.fill(off); err != nil {
				return n, err
			}
		}
		c := copy(p[n:], w.buf[off-w.off:])
		n += c
		off += int64(c)
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (w *windowed) fill(off int64) error {
	size := int64(w.win)
	if off+size > w.size {
		size = w.size - off
	}
	if cap(w.buf) < int(size) {
		w.buf = make([]byte, size)
	}
	w.buf = w.buf[:size]
	n, err := w.ra.ReadAt(w.buf, off)
	if err != nil && !(err == io.EOF && int64(n) == size) {
		w.buf, w.off = w.buf[:0], -1
		return err
	}
	w.off = off
	return nil
}

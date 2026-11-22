// Package collect oferece um Sink e um Source em memória para testes de tradutores.
package collect

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// Sink guarda tudo que recebe. Reject, se definido, descarta atividades.
type Sink struct {
	Activities []*activity.Activity
	ByActivity map[activity.ID][]activity.Sample
	Reject     func(*activity.Activity) bool
}

func NewSink() *Sink { return &Sink{ByActivity: map[activity.ID][]activity.Sample{}} }

func (s *Sink) Activity(_ context.Context, a *activity.Activity) (bool, error) {
	if s.Reject != nil && s.Reject(a) {
		return false, nil
	}
	s.Activities = append(s.Activities, a)
	return true, nil
}

func (s *Sink) Samples(_ context.Context, id activity.ID, b []activity.Sample) error {
	s.ByActivity[id] = append(s.ByActivity[id], b...)
	return nil
}

// Source é um arquivo em memória.
type Source struct{ *bytes.Reader }

func Bytes(b []byte) Source { return Source{bytes.NewReader(b)} }

func (s Source) Size() int64 { return s.Reader.Size() }

// File lê um arquivo de testdata.
func File(t testing.TB, path string) Source {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return Bytes(b)
}

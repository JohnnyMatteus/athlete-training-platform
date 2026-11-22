//go:build load

// Package load é o teste de carga da issue M1-14. Roda só no CI (job manual) ou à mão:
//
//	go test -tags=load -timeout 40m -v ./test/load
//
// Variáveis: LOAD_SIZES_MB (padrão "100,500,1024"), LOAD_DATABASE_URL (opcional; sem ela o
// worker grava num destino nulo, medindo tradução, deduplicação e montagem dos lotes).
package load

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/applehealth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/ingest"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/memprobe"
)

const (
	memoryMargin = 48 << 20 // variação máxima do pico entre o menor e o maior export
	sloMB        = 500
	sloLimit     = 5 * time.Minute
)

func sizes() []int64 {
	raw := os.Getenv("LOAD_SIZES_MB")
	if raw == "" {
		raw = "100,500,1024"
	}
	var out []int64
	for _, s := range strings.Split(raw, ",") {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

func generate(t *testing.T, dir string, mb, seed int64) (path, sum string) {
	t.Helper()
	path = filepath.Join(dir, fmt.Sprintf("export-%dmb.zip", mb))
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	h := sha256.New()
	start := time.Now()
	require.NoError(t, fixtures.WriteAppleExport(io.MultiWriter(f, h), fixtures.AppleOptions{Seed: seed, TargetXMLBytes: mb << 20}))
	st, _ := f.Stat()
	t.Logf("gerado %d MB de XML em %s (%d MB de zip)", mb, time.Since(start).Round(time.Second), st.Size()>>20)
	return path, hex.EncodeToString(h.Sum(nil))
}

// nullDB aceita tudo e conta; permite medir o worker sem banco.
type nullDB struct{ samples, acts int }

func (d *nullDB) Do(ctx context.Context, fn func(context.Context, port.Tx) error) error {
	return fn(ctx, d)
}
func (d *nullDB) FindPrimaries(context.Context, port.AthleteID, time.Time, time.Time) ([]activity.Candidate, error) {
	return nil, nil
}
func (d *nullDB) InsertActivity(context.Context, *activity.Activity) (bool, error) {
	d.acts++
	return true, nil
}
func (d *nullDB) AddAlternative(context.Context, activity.ID, port.Alternative) (bool, error) {
	return true, nil
}
func (d *nullDB) Supersede(context.Context, activity.ID, []activity.Candidate) error { return nil }
func (d *nullDB) CopySamples(_ context.Context, rows []port.SampleRow) error {
	d.samples += len(rows)
	return nil
}
func (d *nullDB) MarkProcessed(context.Context, uuid.UUID) (bool, error) { return true, nil }
func (d *nullDB) Enqueue(context.Context, ...port.Event) error           { return nil }

type fileStore struct{ path string }

type fileObj struct {
	*os.File
	size int64
}

func (o fileObj) Size() int64 { return o.size }

func (f fileStore) Open(context.Context, string, string) (port.Object, error) {
	h, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	st, _ := h.Stat()
	return fileObj{h, st.Size()}, nil
}

func TestConstantMemoryAndSLO(t *testing.T) {
	dir := t.TempDir()
	var peaks []uint64
	for _, mb := range sizes() {
		path, _ := generate(t, dir, mb, 42)
		db := &nullDB{}
		uc := ingest.New(fileStore{path}, db, map[string]port.Translator{"zip": applehealth.Translator{}}, ingest.Config{}, nil)

		start := time.Now()
		peak, err := memprobe.Peak(func() error {
			return uc.Handle(context.Background(), port.ImportReceived{
				EventID: uuid.New(), ImportID: uuid.New(), AthleteID: uuid.New(), FileType: "zip", Bucket: "b", ObjectKey: "k", Attempt: 1})
		})
		elapsed := time.Since(start)
		require.NoError(t, err)
		t.Logf("RESULTADO %d MB: pico de heap %d MB, %s, %d atividades, %d amostras", mb, peak>>20, elapsed.Round(time.Second), db.acts, db.samples)
		assert.Positive(t, db.samples)
		peaks = append(peaks, peak)
		if mb == sloMB {
			assert.Less(t, elapsed, sloLimit, "SLO: export de %d MB em menos de 5 min", mb)
		}
		require.NoError(t, os.Remove(path))
	}
	lo, hi := peaks[0], peaks[0]
	for _, p := range peaks {
		lo, hi = min(lo, p), max(hi, p)
	}
	assert.Less(t, hi-lo, uint64(memoryMargin), "pico de memória deve variar menos que %d MB entre os tamanhos", memoryMargin>>20)
}

func TestGeneratorIsReproducible(t *testing.T) {
	dir := t.TempDir()
	_, a := generate(t, dir, 20, 7)
	_, b := generate(t, dir, 20, 7)
	_, c := generate(t, dir, 20, 8)
	assert.Equal(t, a, b, "mesma semente, mesmo sha256")
	assert.NotEqual(t, a, c)
}

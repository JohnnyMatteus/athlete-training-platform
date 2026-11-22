//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/applehealth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/fit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/postgres"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/ingest"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/collect"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
)

type fileStore map[string][]byte

type obj struct {
	collect.Source
}

func (obj) Close() error { return nil }

func (f fileStore) Open(_ context.Context, _, key string) (port.Object, error) {
	return obj{collect.Bytes(f[key])}, nil
}

func setup(t *testing.T) (*pgxpool.Pool, *ingest.IngestImport, fileStore) {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "timescale/timescaledb:2.17.2-pg16",
		tcpg.WithDatabase("atividades"), tcpg.WithUsername("atp"), tcpg.WithPassword("dev-only-change-me"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// as migrações da fundação são a fonte do esquema
	files, err := filepath.Glob("../../../../../infra/db/migrations/atividades/*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	sort.Strings(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, f)
	}

	store := fileStore{}
	uc := ingest.New(store, postgres.NewStore(pool), map[string]port.Translator{
		"zip": applehealth.Translator{}, "fit": fit.Translator{}}, ingest.Config{BatchSize: 500}, nil)
	return pool, uc, store
}

func msg(athlete uuid.UUID, fileType, key string) port.ImportReceived {
	return port.ImportReceived{EventID: uuid.New(), ImportID: uuid.New(), AthleteID: athlete, FileType: fileType,
		Bucket: "uploads", ObjectKey: key, Attempt: 1}
}

func count(t *testing.T, pool *pgxpool.Pool, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), q, args...).Scan(&n))
	return n
}

func TestImportReimportAndEvents(t *testing.T) {
	pool, uc, store := setup(t)
	ctx := context.Background()
	athlete := uuid.New()
	store["apple"] = fixtures.AppleReduced()

	require.NoError(t, uc.Handle(ctx, msg(athlete, "zip", "apple")))
	assert.Equal(t, 3, count(t, pool, `SELECT count(*) FROM atividade WHERE atleta_id=$1`, athlete))
	assert.Greater(t, count(t, pool, `SELECT count(*) FROM amostra`), 1000)
	assert.Equal(t, 3, count(t, pool, `SELECT count(*) FROM outbox WHERE tipo='TreinoImportado'`))
	assert.Equal(t, 1, count(t, pool, `SELECT count(*) FROM outbox WHERE tipo='ImportacaoConcluida'`))

	before := count(t, pool, `SELECT count(*) FROM amostra`)
	require.NoError(t, uc.Handle(ctx, msg(athlete, "zip", "apple")))
	assert.Equal(t, 3, count(t, pool, `SELECT count(*) FROM atividade`))
	assert.Equal(t, before, count(t, pool, `SELECT count(*) FROM amostra`))
	assert.Equal(t, 3, count(t, pool, `SELECT count(*) FROM outbox WHERE tipo='TreinoImportado'`))
	assert.Equal(t, 3, count(t, pool, `SELECT (payload->'dados'->>'duplicadas')::int FROM outbox
		WHERE tipo='ImportacaoConcluida' ORDER BY id DESC LIMIT 1`))
}

func TestSameEventTwiceProcessesOnce(t *testing.T) {
	pool, uc, store := setup(t)
	store["apple"] = fixtures.AppleReduced()
	m := msg(uuid.New(), "zip", "apple")
	require.NoError(t, uc.Handle(context.Background(), m))
	require.NoError(t, uc.Handle(context.Background(), m))
	assert.Equal(t, 1, count(t, pool, `SELECT count(*) FROM outbox WHERE tipo='ImportacaoConcluida'`))
}

func TestCrossSourceDedupFitReplacesAppleCycling(t *testing.T) {
	pool, uc, store := setup(t)
	ctx := context.Background()
	athlete := uuid.New()
	store["apple"] = fixtures.AppleReduced() // pedal do dia seguinte, só FC no relógio

	// o pedal do export está em Base+24h; alinhamos o .fit a ele
	store["fit"] = fixtures.FitBikePowerAt(fixtures.Base.Add(24*time.Hour + 2*time.Minute))

	require.NoError(t, uc.Handle(ctx, msg(athlete, "zip", "apple")))
	require.NoError(t, uc.Handle(ctx, msg(athlete, "fit", "fit")))

	assert.Equal(t, 3, count(t, pool, `SELECT count(*) FROM atividade`))
	var fonte string
	var alts int
	require.NoError(t, pool.QueryRow(ctx, `SELECT fonte, jsonb_array_length(metadados->'fontes_alternativas')
		FROM atividade WHERE tipo='ciclismo_outdoor'`).Scan(&fonte, &alts))
	assert.Equal(t, "fit", fonte)
	assert.Equal(t, 1, alts)
}

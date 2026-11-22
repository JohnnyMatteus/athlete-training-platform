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

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/postgres"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// Um Postgres só hospeda os dois bancos lógicos do teste: metricas (migração real) e as amostras (esquema mínimo).
func setup(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "timescale/timescaledb:2.17.2-pg16",
		tcpg.WithDatabase("metricas"), tcpg.WithUsername("atp"), tcpg.WithPassword("dev-only-change-me"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// todas as migrações do banco, em ordem: a fonte do esquema é o diretório, não um arquivo
	files, err := filepath.Glob("../../../../../infra/db/migrations/metricas/*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	sort.Strings(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, f)
	}
	return pool
}

type fixedSamples []load.Sample

func (f fixedSamples) Samples(context.Context, uuid.UUID) ([]load.Sample, error) { return f, nil }

func TestImportAndCorrectionAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()
	pool := setup(t)
	athlete, a1, a2 := uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO limiar VALUES ($1, 'ftp', 250, '2026-01-01'), ($1, 'ftp', 500, '2026-04-01')`, athlete)
	require.NoError(t, err)

	start := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	var samples fixedSamples
	for i := range 3600 {
		w := 200
		samples = append(samples, load.Sample{At: start.Add(time.Duration(i) * time.Second), Power: &w})
	}
	svc := metrics.NewService(postgres.NewStore(pool), samples, time.Now)
	ev := func(id uuid.UUID, d int) metrics.ActivityEvent {
		return metrics.ActivityEvent{EventID: uuid.New(), AthleteID: athlete, ActivityID: id, Type: "ciclismo_outdoor",
			Start: start.AddDate(0, 0, d), DurationS: 3600}
	}
	first := ev(a1, 0)
	require.NoError(t, svc.OnActivityImported(ctx, first))
	require.NoError(t, svc.OnActivityImported(ctx, first)) // reentrega
	require.NoError(t, svc.OnActivityImported(ctx, ev(a2, 2)))

	var rows int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM carga_diaria WHERE atleta_id=$1`, athlete).Scan(&rows))
	assert.Equal(t, 3, rows)
	var events int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM outbox`).Scan(&events))
	assert.Equal(t, 2, events, "a reentrega não gera outro MetricasCalculadas")

	var tss, ctl float64
	require.NoError(t, pool.QueryRow(ctx, `SELECT tss::float8, ctl::float8 FROM carga_diaria WHERE atleta_id=$1 AND dia='2026-03-03'`, athlete).Scan(&tss, &ctl))
	assert.InDelta(t, 64, tss, 1e-9) // FTP 250 vigente em março, não os 500 de abril

	// Corrigir a primeira atividade recalcula do dia dela em diante, sem mudar o que já era certo.
	fix := ev(a1, 0)
	fix.DurationS = 1800
	short := metrics.NewService(postgres.NewStore(pool), samples[:1800], time.Now)
	require.NoError(t, short.OnActivityCorrected(ctx, fix))
	var after float64
	require.NoError(t, pool.QueryRow(ctx, `SELECT ctl::float8 FROM carga_diaria WHERE atleta_id=$1 AND dia='2026-03-03'`, athlete).Scan(&after))
	assert.Less(t, after, ctl)
}

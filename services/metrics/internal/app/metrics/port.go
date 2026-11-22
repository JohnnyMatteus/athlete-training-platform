// Package metrics orquestra o cálculo de carga: lê amostras, aplica o domínio e persiste de forma idempotente.
package metrics

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

const (
	ExchangeMetrics      = "metricas"
	RoutingKeyCalculated = "metricas.calculadas"
	EventCalculated      = "MetricasCalculadas"
)

// SampleSource lê as amostras de uma atividade (ver ADR 0011 sobre a origem dos dados).
type SampleSource interface {
	Samples(ctx context.Context, activityID uuid.UUID) ([]load.Sample, error)
}

// UnitOfWork roda fn numa transação; erro faz rollback de tudo, inclusive da inbox.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(context.Context, Tx) error) error
}

// ActivityLoad é a linha de carga_atividade.
type ActivityLoad struct {
	ActivityID uuid.UUID
	AthleteID  uuid.UUID
	Type       string
	Result     load.Result
}

// OutEvent é uma linha do outbox.
type OutEvent struct {
	EventID     uuid.UUID
	AggregateID uuid.UUID
	RoutingKey  string
	Type        string
	Version     int
	Payload     []byte
}

// Tx é o que o caso de uso precisa do banco `metricas`.
type Tx interface {
	// LockAthlete serializa recálculos do mesmo atleta até o fim da transação.
	LockAthlete(ctx context.Context, athlete uuid.UUID) error
	// MarkProcessed grava o event_id na inbox; false se já tinha sido processado.
	MarkProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	Thresholds(ctx context.Context, athlete uuid.UUID) (load.ThresholdHistory, error)
	// Profile devolve o perfil do atleta; zero quando não cadastrado.
	Profile(ctx context.Context, athlete uuid.UUID) (load.Athlete, error)
	SaveActivityLoad(ctx context.Context, l ActivityLoad) error
	// DailyTSS soma o TSS das atividades por dia, de from em diante.
	DailyTSS(ctx context.Context, athlete uuid.UUID, from load.Day) (map[load.Day]float64, error)
	// LastDay é o último dia de carga_diaria do atleta.
	LastDay(ctx context.Context, athlete uuid.UUID) (load.Day, bool, error)
	// StateBefore devolve a última linha de carga_diaria anterior a day.
	StateBefore(ctx context.Context, athlete uuid.UUID, day load.Day) (load.Day, load.State, bool, error)
	// UpsertDaily grava as linhas por (atleta, dia).
	UpsertDaily(ctx context.Context, athlete uuid.UUID, rows []load.DayLoad) error
	Enqueue(ctx context.Context, e OutEvent) error
}

// Clock isola o relógio nos testes.
type Clock func() time.Time

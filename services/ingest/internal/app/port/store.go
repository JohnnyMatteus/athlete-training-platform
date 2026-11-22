package port

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

type AthleteID = uuid.UUID

// Object é o arquivo no armazenamento de objetos.
type Object interface {
	Source
	io.Closer
}

// ObjectStore lê arquivos de importação (MinIO/S3).
type ObjectStore interface {
	Open(ctx context.Context, bucket, key string) (Object, error)
}

// Alternative é uma FonteAlternativa: o mesmo treino registrado por outra fonte.
type Alternative struct {
	Key       string
	Source    activity.Source
	Start     time.Time
	DurationS int
	Channels  activity.Channels
}

// SampleRow é uma linha de amostra pronta para escrita em lote.
type SampleRow struct {
	ActivityID activity.ID
	activity.Sample
}

// Tx é o conjunto de operações transacionais da ingestão.
type Tx interface {
	// FindPrimaries lista as atividades principais do atleta que tocam [from, to].
	FindPrimaries(ctx context.Context, athlete AthleteID, from, to time.Time) ([]activity.Candidate, error)
	// InsertActivity faz upsert pela chave natural; inserted=false significa já existente.
	InsertActivity(ctx context.Context, a *activity.Activity) (inserted bool, err error)
	// AddAlternative registra a fonte alternativa; added=false se já estava registrada.
	AddAlternative(ctx context.Context, primary activity.ID, alt Alternative) (added bool, err error)
	// Supersede remove as principais antigas (e suas amostras) e as registra, com as
	// alternativas que já carregavam, como fontes alternativas da nova.
	Supersede(ctx context.Context, newID activity.ID, old []activity.Candidate) error
	// CopySamples grava amostras em lote (COPY), nunca linha a linha.
	CopySamples(ctx context.Context, rows []SampleRow) error
	// MarkProcessed grava o event_id na inbox; first=false se a mensagem já foi processada.
	MarkProcessed(ctx context.Context, eventID uuid.UUID) (first bool, err error)
	// Enqueue grava eventos na outbox, na mesma transação.
	Enqueue(ctx context.Context, events ...Event) error
}

// UnitOfWork executa fn numa transação; erro faz rollback.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

// Totals são os totais do lote. Alternativas (mesmo treino de outra fonte) contam como duplicadas.
type Totals struct {
	Imported   int
	Duplicated int
	Errors     int
	Superseded int
}

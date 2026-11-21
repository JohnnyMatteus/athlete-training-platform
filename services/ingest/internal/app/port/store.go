package port

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

type AthleteID = uuid.UUID

// ErrImportNotFound indica lote inexistente.
var ErrImportNotFound = errors.New("importação não encontrada")

// Import é o lote de importação (lido do contexto Importação).
type Import struct {
	ID        uuid.UUID
	AthleteID AthleteID
	Format    activity.Source // apple_export, fit, tcx ou gpx
	ObjectKey string
	Attempts  int
}

// Object é o arquivo no armazenamento de objetos.
type Object interface {
	Source
	io.Closer
}

// ObjectStore lê arquivos de importação (MinIO/S3).
type ObjectStore interface {
	Open(ctx context.Context, key string) (Object, error)
}

// ImportStore controla o ciclo de vida do lote, fora da transação de ingestão.
type ImportStore interface {
	// Begin reserva o lote para processamento e incrementa as tentativas.
	// Devolve started=false se o lote já terminou (concluído ou falho): a mensagem é repetida.
	Begin(ctx context.Context, importID uuid.UUID) (imp Import, started bool, err error)
	// Fail marca o lote como falho e enfileira ImportacaoFalhou, atomicamente.
	Fail(ctx context.Context, importID uuid.UUID, reason string, attempts int) error
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
	// CompleteImport marca o lote como concluído com os totais.
	CompleteImport(ctx context.Context, importID uuid.UUID, t Totals) error
	// Enqueue grava eventos na outbox, na mesma transação.
	Enqueue(ctx context.Context, events ...Event) error
}

// UnitOfWork executa fn numa transação; erro faz rollback.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

// Totals são os totais do lote.
type Totals struct {
	Imported    int `json:"importadas"`
	Duplicated  int `json:"duplicadas"`
	Errors      int `json:"erros"`
	Superseded  int `json:"substituidas"`
	Alternative int `json:"alternativas"`
}

package port

import (
	"time"

	"github.com/google/uuid"
)

// Event é um evento de integração a publicar (outbox → RabbitMQ).
// Os nomes e campos seguem contracts/events/importacao (versão 1).
type Event struct {
	ID         uuid.UUID
	Type       string
	OccurredAt time.Time
	Payload    any
}

const (
	EventImportReceived  = "ImportacaoRecebida"
	EventActivityImport  = "TreinoImportado"
	EventImportDone      = "ImportacaoConcluida"
	EventImportFailed    = "ImportacaoFalhou"
	EventActivityCorrect = "AtividadeCorrigida"
)

// ImportReceived é a mensagem consumida.
type ImportReceived struct {
	EventID  uuid.UUID `json:"event_id"`
	ImportID uuid.UUID `json:"import_id"`
}

// ActivityImported: uma por atividade nova.
type ActivityImported struct {
	ImportID   uuid.UUID `json:"import_id"`
	ActivityID uuid.UUID `json:"atividade_id"`
	AthleteID  uuid.UUID `json:"atleta_id"`
	Type       string    `json:"tipo"`
	Source     string    `json:"fonte"`
	StartUTC   time.Time `json:"inicio_utc"`
}

// ImportDone: totais do lote.
type ImportDone struct {
	ImportID uuid.UUID `json:"import_id"`
	Totals
}

// ImportFailed: lote falho depois do limite de tentativas ou por arquivo malformado.
type ImportFailed struct {
	ImportID uuid.UUID `json:"import_id"`
	Reason   string    `json:"motivo"`
	Attempts int       `json:"tentativas"`
}

// ActivityCorrected: a atividade principal foi trocada por outra fonte melhor.
type ActivityCorrected struct {
	ActivityID   uuid.UUID `json:"atividade_id"`
	SupersededBy uuid.UUID `json:"substituida_por"`
}

package port

import (
	"time"

	"github.com/google/uuid"
)

// Event é um evento de integração a publicar (outbox → RabbitMQ), versão 1 dos contratos em
// contracts/events/importacao. Payload é o objeto "dados"; o envelope é montado na outbox.
type Event struct {
	ID         uuid.UUID
	Type       string
	RoutingKey string
	ImportID   uuid.UUID
	OccurredAt time.Time
	Payload    any
}

const (
	EventImportReceived   = "ImportacaoRecebida"
	EventActivityImported = "TreinoImportado"
	EventImportDone       = "ImportacaoConcluida"
	EventImportFailed     = "ImportacaoFalhou"

	RouteActivityImported = "treino.importado"
	RouteImportDone       = "importacao.concluida"
	RouteImportFailed     = "importacao.falhou"
)

// Códigos de falha do contrato ImportacaoFalhou.
const (
	CodeInvalidFile       = "arquivo_invalido"
	CodeUnsupportedFormat = "formato_nao_suportado"
	CodeInternal          = "falha_interna"
)

// ImportReceived é a mensagem ImportacaoRecebida já decodificada.
type ImportReceived struct {
	EventID   uuid.UUID
	ImportID  uuid.UUID
	AthleteID uuid.UUID
	FileType  string // zip, fit, tcx ou gpx
	Bucket    string
	ObjectKey string
	SizeBytes int64
	SHA256    string
	Attempt   int // entrega atual, começando em 1
}

// ActivityImported é o "dados" de TreinoImportado.
type ActivityImported struct {
	AthleteID  uuid.UUID `json:"atleta_id"`
	ActivityID uuid.UUID `json:"atividade_id"`
	Type       string    `json:"tipo"`
	Source     string    `json:"fonte"`
	StartUTC   time.Time `json:"inicio_utc"`
	DurationS  int       `json:"duracao_s"`
	DistanceM  *float64  `json:"distancia_m"`
	Samples    int       `json:"quantidade_amostras"`
}

// ImportDone é o "dados" de ImportacaoConcluida.
type ImportDone struct {
	AthleteID     uuid.UUID `json:"atleta_id"`
	Imported      int       `json:"importadas"`
	Duplicated    int       `json:"duplicadas"`
	Errors        int       `json:"erros"`
	FormatVersion *string   `json:"versao_formato"`
}

// ImportFailed é o "dados" de ImportacaoFalhou.
type ImportFailed struct {
	AthleteID uuid.UUID `json:"atleta_id"`
	Code      string    `json:"codigo"`
	Reason    string    `json:"motivo"`
	Attempts  int       `json:"tentativas"`
}

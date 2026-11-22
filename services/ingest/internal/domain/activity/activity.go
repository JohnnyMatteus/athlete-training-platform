package activity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ID identifica uma atividade.
type ID = uuid.UUID

// Activity é o agregado raiz do contexto Atividades.
// As amostras não fazem parte do estado do agregado: podem ser milhões e trafegam em lotes.
type Activity struct {
	ID        ID
	AthleteID uuid.UUID
	Type      Type
	Source    Source
	Device    string
	Start     time.Time
	DurationS int
	DistanceM *float64
	Channels  Channels
	Laps      []Lap
	Route     []RoutePoint
	Metadata  map[string]any

	events []Event
}

// Params reúne os dados brutos de uma atividade para New.
type Params struct {
	AthleteID    uuid.UUID
	Type         Type
	OriginalType string // tipo de terceiros; preservado nos metadados quando Type é Outro
	Source       Source
	Device       string
	Start        Instant
	DurationS    int
	DistanceM    *float64
	Channels     Channels
	Laps         []Lap
	Route        []RoutePoint
	Metadata     map[string]any
}

var ErrInvalidActivity = errors.New("atividade inválida")

// New cria uma atividade nova e registra AtividadeRegistrada.
func New(p Params) (*Activity, error) {
	if !p.Source.Valid() {
		return nil, fmt.Errorf("%w: fonte %q", ErrInvalidActivity, p.Source)
	}
	if p.DurationS < 0 {
		return nil, fmt.Errorf("%w: duração negativa", ErrInvalidActivity)
	}
	if p.DistanceM != nil && *p.DistanceM < 0 {
		return nil, fmt.Errorf("%w: distância negativa", ErrInvalidActivity)
	}
	if p.Type.Kind == "" {
		p.Type = TypeOther
	}
	meta := make(map[string]any, len(p.Metadata)+1)
	for k, v := range p.Metadata {
		meta[k] = v
	}
	if p.Type.Kind == KindOther && p.OriginalType != "" {
		meta["tipo_original"] = p.OriginalType
	}
	a := &Activity{
		ID: uuid.New(), AthleteID: p.AthleteID, Type: p.Type, Source: p.Source, Device: p.Device,
		Start: p.Start.Time(), DurationS: p.DurationS, DistanceM: p.DistanceM, Channels: p.Channels,
		Laps: p.Laps, Route: p.Route, Metadata: meta,
	}
	a.events = append(a.events, ActivityRegistered{ActivityID: a.ID, AthleteID: a.AthleteID, Source: a.Source, Type: a.Type, Start: a.Start})
	return a, nil
}

// End é o instante final da atividade.
func (a *Activity) End() time.Time { return a.Start.Add(time.Duration(a.DurationS) * time.Second) }

// NaturalKey identifica a atividade de forma determinística:
// sha256(fonte, inicio_utc, duracao, tipo, dispositivo). Reimportar o mesmo arquivo gera a mesma chave.
func (a *Activity) NaturalKey() string {
	return NaturalKey(a.Source, a.Start, a.DurationS, a.Type, a.Device)
}

func NaturalKey(src Source, start time.Time, durationS int, t Type, device string) string {
	h := sha256.New()
	// separador \x1f evita colisão entre campos adjacentes
	h.Write([]byte(strings.Join([]string{
		string(src), strconv.FormatInt(start.UTC().UnixNano(), 10), strconv.Itoa(durationS), t.String(), device,
	}, "\x1f")))
	return hex.EncodeToString(h.Sum(nil))
}

// PullEvents devolve e limpa os eventos de domínio pendentes.
func (a *Activity) PullEvents() []Event {
	ev := a.events
	a.events = nil
	return ev
}

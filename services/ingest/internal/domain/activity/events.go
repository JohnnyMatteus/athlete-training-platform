package activity

import (
	"time"

	"github.com/google/uuid"
)

// Event é um evento de domínio do contexto Atividades.
type Event interface{ EventName() string }

// ActivityRegistered corresponde a AtividadeRegistrada.
type ActivityRegistered struct {
	ActivityID uuid.UUID
	AthleteID  uuid.UUID
	Source     Source
	Type       Type
	Start      time.Time
}

func (ActivityRegistered) EventName() string { return "AtividadeRegistrada" }

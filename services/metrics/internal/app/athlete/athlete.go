// Package athlete reúne os casos de uso de leitura de métricas, perfil, limiares e o read model de vínculos (ADR 0014).
package athlete

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

var (
	// ErrNotFound vale também para atleta que não consentiu: não revela existência.
	ErrNotFound  = errors.New("não encontrado")
	ErrForbidden = errors.New("papel sem permissão")
	ErrInvalid   = errors.New("entrada inválida")
)

const (
	RoleAthlete = "atleta"
	RoleCoach   = "treinador"

	// Categorias de consentimento exigidas de um treinador: carga e atividades exigem `treinos`;
	// perfil (FC) e limiares exigem `frequencia_cardiaca`.
	CategoryTraining   = "treinos"
	CategoryHeartRate  = "frequencia_cardiaca"
	ExchangeMetrics    = "metricas"
	RoutingKeyLimiar   = "limiar.atualizado"
	EventLimiarUpdated = "LimiarAtualizado"
)

// Principal é o chamador autenticado.
type Principal struct {
	ID    uuid.UUID
	Roles []string
}

func (p Principal) Has(role string) bool { return slices.Contains(p.Roles, role) }

type DayLoad struct {
	Day                string
	TSS, CTL, ATL, TSB float64
}

type ActivityRow struct {
	ActivityID                  uuid.UUID
	Day, Type, Method           string
	TSS                         float64
	NP, IntensityFactor         *float64
	DiscardedHR, DiscardedPower int
}

type Threshold struct {
	Kind      string
	Value     float64
	ValidFrom string
}

type Profile struct {
	Sex       string
	RestingHR int
	MaxHR     int
}

// Link é o vínculo recebido por evento.
type Link struct {
	ID, AthleteID, CoachID uuid.UUID
	Categories             []string
	Active                 bool
	At                     time.Time
}

// OutEvent é uma linha do outbox.
type OutEvent struct {
	EventID, AggregateID uuid.UUID
	RoutingKey, Type     string
	Payload              []byte
}

// Store é a porta de persistência do banco `metricas` usada por este pacote.
type Store interface {
	DailyLoad(ctx context.Context, athlete uuid.UUID, from, to load.Day) ([]DayLoad, error)
	Activities(ctx context.Context, athlete uuid.UUID, from, to load.Day) ([]ActivityRow, error)
	Thresholds(ctx context.Context, athlete uuid.UUID) ([]Threshold, error)
	Profile(ctx context.Context, athlete uuid.UUID) (Profile, bool, error)
	SaveProfile(ctx context.Context, athlete uuid.UUID, p Profile) error
	// SaveThreshold grava o limiar e o evento na mesma transação.
	SaveThreshold(ctx context.Context, athlete uuid.UUID, t Threshold, ev OutEvent) error
	HasActiveLink(ctx context.Context, athlete, coach uuid.UUID, category string) (bool, error)
	// ApplyLink grava o vínculo e marca o evento como processado; false se já processado ou se o estado guardado é mais novo.
	ApplyLink(ctx context.Context, eventID uuid.UUID, l Link) (bool, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(s Store, now func() time.Time) *Service { return &Service{store: s, now: now} }

// target resolve de quem são os dados lidos. Treinador só alcança atleta com vínculo ativo e categoria consentida.
func (s *Service) target(ctx context.Context, p Principal, athlete *uuid.UUID, category string) (uuid.UUID, error) {
	if athlete == nil || *athlete == p.ID {
		return p.ID, nil
	}
	if !p.Has(RoleCoach) {
		return uuid.Nil, ErrNotFound
	}
	ok, err := s.store.HasActiveLink(ctx, *athlete, p.ID, category)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, ErrNotFound
	}
	return *athlete, nil
}

func (s *Service) window(from, to *load.Day) (load.Day, load.Day) {
	end := load.DayOf(s.now())
	start := end - 90
	if to != nil {
		end = *to
	}
	if from != nil {
		start = *from
	}
	return start, end
}

func (s *Service) Load(ctx context.Context, p Principal, athlete *uuid.UUID, from, to *load.Day) (uuid.UUID, []DayLoad, error) {
	id, err := s.target(ctx, p, athlete, CategoryTraining)
	if err != nil {
		return uuid.Nil, nil, err
	}
	a, b := s.window(from, to)
	if a > b {
		return uuid.Nil, nil, fmt.Errorf("%w: de depois de ate", ErrInvalid)
	}
	rows, err := s.store.DailyLoad(ctx, id, a, b)
	return id, rows, err
}

func (s *Service) Activities(ctx context.Context, p Principal, athlete *uuid.UUID, from, to *load.Day) ([]ActivityRow, error) {
	id, err := s.target(ctx, p, athlete, CategoryTraining)
	if err != nil {
		return nil, err
	}
	a, b := s.window(from, to)
	if a > b {
		return nil, fmt.Errorf("%w: de depois de ate", ErrInvalid)
	}
	return s.store.Activities(ctx, id, a, b)
}

func (s *Service) Thresholds(ctx context.Context, p Principal, athlete *uuid.UUID) ([]Threshold, error) {
	id, err := s.target(ctx, p, athlete, CategoryHeartRate)
	if err != nil {
		return nil, err
	}
	return s.store.Thresholds(ctx, id)
}

func (s *Service) Profile(ctx context.Context, p Principal, athlete *uuid.UUID) (Profile, error) {
	id, err := s.target(ctx, p, athlete, CategoryHeartRate)
	if err != nil {
		return Profile{}, err
	}
	pr, ok, err := s.store.Profile(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	if !ok {
		return Profile{}, ErrNotFound
	}
	return pr, nil
}

// PutProfile é só do atleta: treinador nunca escreve dado de saúde.
func (s *Service) PutProfile(ctx context.Context, p Principal, pr Profile) error {
	if !p.Has(RoleAthlete) {
		return ErrForbidden
	}
	if (pr.Sex != "M" && pr.Sex != "F") || pr.RestingHR < 30 || pr.RestingHR > 120 ||
		pr.MaxHR < 100 || pr.MaxHR > 230 || pr.MaxHR <= pr.RestingHR {
		return fmt.Errorf("%w: perfil fora das faixas", ErrInvalid)
	}
	return s.store.SaveProfile(ctx, p.ID, pr)
}

// PutThreshold grava o limiar e publica LimiarAtualizado (outbox).
func (s *Service) PutThreshold(ctx context.Context, p Principal, t Threshold) error {
	if !p.Has(RoleAthlete) {
		return ErrForbidden
	}
	if t.Kind != "ftp" && t.Kind != "pace_limiar" && t.Kind != "css" {
		return fmt.Errorf("%w: tipo desconhecido", ErrInvalid)
	}
	if t.Value <= 0 {
		return fmt.Errorf("%w: valor deve ser positivo", ErrInvalid)
	}
	if _, err := load.ParseDay(t.ValidFrom); err != nil {
		return fmt.Errorf("%w: vigente_desde", ErrInvalid)
	}
	id := uuid.New()
	body, err := json.Marshal(map[string]any{
		"event_id": id, "tipo": EventLimiarUpdated, "versao": 1, "ocorrido_em": s.now().UTC(),
		"dados": map[string]any{"atleta_id": p.ID, "tipo": t.Kind, "valor": t.Value, "vigente_desde": t.ValidFrom},
	})
	if err != nil {
		return err
	}
	return s.store.SaveThreshold(ctx, p.ID, t, OutEvent{EventID: id, AggregateID: p.ID, RoutingKey: RoutingKeyLimiar, Type: EventLimiarUpdated, Payload: body})
}

// OnLink aplica VinculoTreinadorConsentido ou Revogado ao read model.
func (s *Service) OnLink(ctx context.Context, eventID uuid.UUID, l Link) error {
	_, err := s.store.ApplyLink(ctx, eventID, l)
	return err
}

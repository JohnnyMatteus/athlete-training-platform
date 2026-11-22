// Package consumer é o adaptador de entrada: converte mensagens do RabbitMQ em chamadas ao caso de uso.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/telemetry"
)

// Filas definidas em infra/rabbitmq/definitions.json.
const (
	QueueImported  = "metrics.treino-importado"
	QueueCorrected = "metrics.atividade-corrigida"
	QueueLinks     = "metrics.vinculo"
)

// ErrInvalidMessage marca mensagem que nunca será processável (vai para a DLQ).
var ErrInvalidMessage = errors.New("mensagem inválida")

type envelope struct {
	EventID uuid.UUID `json:"event_id"`
	Type    string    `json:"tipo"`
	Version int       `json:"versao"`
	Data    struct {
		AthleteID     uuid.UUID  `json:"atleta_id"`
		ActivityID    uuid.UUID  `json:"atividade_id"`
		Type          string     `json:"tipo"`
		Start         time.Time  `json:"inicio_utc"`
		DurationS     int        `json:"duracao_s"`
		DistanceM     *float64   `json:"distancia_m"`
		PreviousStart *time.Time `json:"inicio_utc_anterior"`
	} `json:"dados"`
}

// Decode valida o envelope e devolve o evento de atividade. wantType é o `tipo` esperado.
func Decode(body []byte, wantType string) (metrics.ActivityEvent, error) {
	var e envelope
	if err := json.Unmarshal(body, &e); err != nil {
		return metrics.ActivityEvent{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	d := e.Data
	if e.Type != wantType || e.Version != 1 || e.EventID == uuid.Nil || d.AthleteID == uuid.Nil ||
		d.ActivityID == uuid.Nil || d.Type == "" || d.Start.IsZero() || d.DurationS < 0 {
		return metrics.ActivityEvent{}, fmt.Errorf("%w: %s com campos obrigatórios ausentes", ErrInvalidMessage, wantType)
	}
	return metrics.ActivityEvent{
		EventID: e.EventID, AthleteID: d.AthleteID, ActivityID: d.ActivityID, Type: d.Type,
		Start: d.Start.UTC(), DurationS: d.DurationS, DistanceM: d.DistanceM, PreviousStart: d.PreviousStart,
	}, nil
}

// Handler trata um corpo de mensagem já roteado por fila.
type Handler func(ctx context.Context, body []byte) error

// Imported e Corrected ligam as mensagens ao caso de uso.
func Imported(s *metrics.Service) Handler {
	return func(ctx context.Context, body []byte) error {
		e, err := Decode(body, "TreinoImportado")
		if err != nil {
			return err
		}
		return s.OnActivityImported(ctx, e)
	}
}

func Corrected(s *metrics.Service) Handler {
	return func(ctx context.Context, body []byte) error {
		e, err := Decode(body, "AtividadeCorrigida")
		if err != nil {
			return err
		}
		return s.OnActivityCorrected(ctx, e)
	}
}

// Consumer consome uma fila com ack manual: só confirma depois do commit.
type Consumer struct {
	ch      *amqp.Channel
	queue   string
	handler Handler
	log     *slog.Logger
}

func New(conn *amqp.Connection, queue string, h Handler, log *slog.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return nil, err
	}
	return &Consumer{ch: ch, queue: queue, handler: h, log: log}, nil
}

// Run bloqueia até o contexto terminar ou o canal fechar.
func (c *Consumer) Run(ctx context.Context) error {
	deliveries, err := c.ch.ConsumeWithContext(ctx, c.queue, "metrics-"+c.queue, false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("canal do RabbitMQ fechado")
			}
			c.dispatch(ctx, d)
		}
	}
}

func (c *Consumer) dispatch(ctx context.Context, d amqp.Delivery) {
	ctx = telemetry.WithTraceparent(ctx, headerString(d.Headers, "traceparent"))
	ctx, span := otel.Tracer("metrics").Start(ctx, c.queue+" process", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()
	err := c.handler(ctx, d.Body)
	if err != nil {
		span.RecordError(err)
	}
	switch {
	case err == nil:
		_ = d.Ack(false)
	case errors.Is(err, ErrInvalidMessage):
		c.log.ErrorContext(ctx, "mensagem descartada", "fila", c.queue, "erro", err)
		_ = d.Nack(false, false) // sem requeue: cai em importacao.dlq
	default:
		c.log.ErrorContext(ctx, "falha ao processar, devolvendo à fila", "fila", c.queue, "erro", err)
		_ = d.Nack(false, true)
	}
}

func headerString(h amqp.Table, key string) string {
	if v, ok := h[key].(string); ok {
		return v
	}
	return ""
}

type linkEnvelope struct {
	EventID    uuid.UUID `json:"event_id"`
	Type       string    `json:"tipo"`
	Version    int       `json:"versao"`
	OccurredAt time.Time `json:"ocorrido_em"`
	Data       struct {
		LinkID     uuid.UUID `json:"vinculo_id"`
		AthleteID  uuid.UUID `json:"atleta_id"`
		CoachID    uuid.UUID `json:"treinador_id"`
		Categories []string  `json:"categorias"`
	} `json:"dados"`
}

// DecodeLink valida VinculoTreinadorConsentido e VinculoTreinadorRevogado.
func DecodeLink(body []byte) (uuid.UUID, athlete.Link, error) {
	var e linkEnvelope
	if err := json.Unmarshal(body, &e); err != nil {
		return uuid.Nil, athlete.Link{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	d := e.Data
	consented := e.Type == "VinculoTreinadorConsentido"
	if (!consented && e.Type != "VinculoTreinadorRevogado") || e.Version != 1 || e.EventID == uuid.Nil || e.OccurredAt.IsZero() ||
		d.LinkID == uuid.Nil || d.AthleteID == uuid.Nil || d.CoachID == uuid.Nil || (consented && len(d.Categories) == 0) {
		return uuid.Nil, athlete.Link{}, fmt.Errorf("%w: evento de vínculo com campos obrigatórios ausentes", ErrInvalidMessage)
	}
	return e.EventID, athlete.Link{ID: d.LinkID, AthleteID: d.AthleteID, CoachID: d.CoachID, Categories: d.Categories, Active: consented, At: e.OccurredAt.UTC()}, nil
}

// Links liga os eventos de vínculo ao read model de autorização.
func Links(s *athlete.Service) Handler {
	return func(ctx context.Context, body []byte) error {
		id, l, err := DecodeLink(body)
		if err != nil {
			return err
		}
		return s.OnLink(ctx, id, l)
	}
}

// Package consumer é o adaptador de entrada: converte mensagens do RabbitMQ em chamadas ao caso de uso.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/telemetry"
)

// Queue é a fila definida em infra/rabbitmq/definitions.json.
const Queue = "ingest.importacao-recebida"

// ErrInvalidMessage marca mensagem que nunca será processável (vai para a DLQ).
var ErrInvalidMessage = errors.New("mensagem ImportacaoRecebida inválida")

type envelope struct {
	EventID  uuid.UUID `json:"event_id"`
	Type     string    `json:"tipo"`
	Version  int       `json:"versao"`
	ImportID uuid.UUID `json:"import_id"`
	Data     struct {
		AthleteID uuid.UUID `json:"atleta_id"`
		FileType  string    `json:"tipo_arquivo"`
		Bucket    string    `json:"bucket"`
		ObjectKey string    `json:"object_key"`
		SizeBytes int64     `json:"tamanho_bytes"`
		SHA256    string    `json:"sha256"`
	} `json:"dados"`
}

// Decode converte o corpo da mensagem. deliveryCount é o x-delivery-count (0 na primeira entrega).
func Decode(body []byte, deliveryCount int64) (port.ImportReceived, error) {
	var e envelope
	if err := json.Unmarshal(body, &e); err != nil {
		return port.ImportReceived{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	if e.Type != port.EventImportReceived || e.Version != 1 || e.EventID == uuid.Nil || e.ImportID == uuid.Nil ||
		e.Data.Bucket == "" || e.Data.ObjectKey == "" || e.Data.FileType == "" {
		return port.ImportReceived{}, fmt.Errorf("%w: campos obrigatórios ausentes", ErrInvalidMessage)
	}
	return port.ImportReceived{
		EventID: e.EventID, ImportID: e.ImportID, AthleteID: e.Data.AthleteID, FileType: e.Data.FileType,
		Bucket: e.Data.Bucket, ObjectKey: e.Data.ObjectKey, SizeBytes: e.Data.SizeBytes, SHA256: e.Data.SHA256,
		Attempt: int(deliveryCount) + 1,
	}, nil
}

// Handler é o caso de uso (ingest.IngestImport).
type Handler interface {
	Handle(ctx context.Context, msg port.ImportReceived) error
}

// Consumer consome a fila com ack manual: só confirma depois do commit.
type Consumer struct {
	ch      *amqp.Channel
	handler Handler
	log     *slog.Logger
}

func New(conn *amqp.Connection, h Handler, log *slog.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil { // um lote por vez: o arquivo pode ser grande
		return nil, err
	}
	return &Consumer{ch: ch, handler: h, log: log}, nil
}

// Run bloqueia até o contexto terminar ou o canal fechar.
func (c *Consumer) Run(ctx context.Context) error {
	deliveries, err := c.ch.ConsumeWithContext(ctx, Queue, "ingest-worker", false, false, false, false, nil)
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
	ctx, span := otel.Tracer("ingest").Start(ctx, "ImportacaoRecebida process", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()
	count, _ := d.Headers["x-delivery-count"].(int64) // quorum queues
	msg, err := Decode(d.Body, count)
	if err != nil {
		c.log.ErrorContext(ctx, "mensagem descartada", "erro", err)
		_ = d.Nack(false, false) // sem requeue: cai em importacao.dlq
		return
	}
	if err := c.handler.Handle(ctx, msg); err != nil {
		span.RecordError(err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func headerString(h amqp.Table, key string) string {
	if v, ok := h[key].(string); ok {
		return v
	}
	return ""
}

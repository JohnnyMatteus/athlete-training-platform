// Package rabbit publica eventos no exchange de métricas e consome os eventos de atividade.
package rabbit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/outbox"
)

const Exchange = "metricas"

// Publisher publica com confirmação do broker (publisher confirms).
type Publisher struct{ ch *amqp.Channel }

var _ outbox.Publisher = (*Publisher)(nil)

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		return nil, err
	}
	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, messageID uuid.UUID, eventType, traceparent string, body []byte) error {
	var headers amqp.Table
	if traceparent != "" {
		headers = amqp.Table{"traceparent": traceparent}
	}
	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, Exchange, routingKey, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent,
		Headers: headers, MessageId: messageID.String(), Type: eventType, Timestamp: time.Now().UTC(), Body: body,
	})
	if err != nil {
		return err
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("broker não confirmou %s", messageID)
	}
	return nil
}

func (p *Publisher) Close() error { return p.ch.Close() }

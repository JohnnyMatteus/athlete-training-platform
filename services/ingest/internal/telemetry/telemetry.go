// Package telemetry configura o OpenTelemetry (ADR 0018) e propaga o contexto W3C pelo outbox e pelo AMQP.
package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var propagator = propagation.TraceContext{}

// Setup liga o exportador OTLP/HTTP só se OTEL_EXPORTER_OTLP_ENDPOINT estiver definido.
// A propagação W3C fica ativa sempre; sem exportador os spans são descartados sem custo.
func Setup(ctx context.Context, defaultService string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagator)
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil
	}
	name := os.Getenv("OTEL_SERVICE_NAME")
	if name == "" {
		name = defaultService
	}
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(name))),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Traceparent devolve o cabeçalho W3C do span corrente, ou vazio.
func Traceparent(ctx context.Context) string {
	c := propagation.MapCarrier{}
	propagator.Inject(ctx, c)
	return c.Get("traceparent")
}

// WithTraceparent devolve um contexto filho do traceparent informado (vazio ou inválido não altera).
func WithTraceparent(ctx context.Context, tp string) context.Context {
	if tp == "" {
		return ctx
	}
	return propagator.Extract(ctx, propagation.MapCarrier{"traceparent": tp})
}

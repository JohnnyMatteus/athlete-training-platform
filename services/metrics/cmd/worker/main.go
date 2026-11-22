// Comando worker: consome TreinoImportado e AtividadeCorrigida, calcula a carga e publica MetricasCalculadas.
// Configuração só por variáveis de ambiente; encerra em SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/outbox"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/postgres"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/rabbit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/consumer"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, log); err != nil {
		log.Error("worker encerrado com erro", "erro", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	metricsURL, err := need("METRICS_DATABASE_URL")
	if err != nil {
		return err
	}
	samplesURL, err := need("ATIVIDADES_READONLY_URL")
	if err != nil {
		return err
	}
	amqpURL, err := need("RABBITMQ_URL")
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, metricsURL)
	if err != nil {
		return fmt.Errorf("postgres metricas: %w", err)
	}
	defer pool.Close()
	samplesPool, err := pgxpool.New(ctx, samplesURL)
	if err != nil {
		return fmt.Errorf("postgres atividades: %w", err)
	}
	defer samplesPool.Close()

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return fmt.Errorf("rabbitmq: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			log.Warn("fechar conexão rabbitmq", "erro", err)
		}
	}()
	pub, err := rabbit.NewPublisher(conn)
	if err != nil {
		return err
	}
	defer func() {
		if err := pub.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			log.Warn("fechar canal de publicação", "erro", err)
		}
	}()

	svc := metrics.NewService(postgres.NewStore(pool), postgres.NewSamples(samplesPool), time.Now)
	imported, err := consumer.New(conn, consumer.QueueImported, consumer.Imported(svc), log)
	if err != nil {
		return err
	}
	corrected, err := consumer.New(conn, consumer.QueueCorrected, consumer.Corrected(svc), log)
	if err != nil {
		return err
	}
	relay := outbox.NewRelay(pool, pub, time.Second, log)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return imported.Run(ctx) })
	g.Go(func() error { return corrected.Run(ctx) })
	g.Go(func() error { relay.Run(ctx); return nil })
	log.Info("serviço de métricas iniciado")
	return g.Wait()
}

func need(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("variável %s não definida", name)
	}
	return v, nil
}

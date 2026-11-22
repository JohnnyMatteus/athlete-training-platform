// Comando worker: consome ImportacaoRecebida, ingere o arquivo e publica os resultados.
// Configuração só por variáveis de ambiente; encerra em SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/applehealth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/fit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/gpx"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/tcx"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/objectstore"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/outbox"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/postgres"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/adapters/rabbit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/ingest"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/interfaces/consumer"
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
	dbURL, err := need("INGEST_DATABASE_URL")
	if err != nil {
		return err
	}
	amqpURL, err := need("RABBITMQ_URL")
	if err != nil {
		return err
	}
	endpoint, err := need("MINIO_ENDPOINT")
	if err != nil {
		return err
	}
	user, pass := os.Getenv("MINIO_ROOT_USER"), os.Getenv("MINIO_ROOT_PASSWORD")

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	mc, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(user, pass, ""), Secure: os.Getenv("MINIO_USE_SSL") == "true",
	})
	if err != nil {
		return fmt.Errorf("minio: %w", err)
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return fmt.Errorf("rabbitmq: %w", err)
	}
	defer conn.Close()

	pub, err := rabbit.NewPublisher(conn)
	if err != nil {
		return err
	}
	uc := ingest.New(objectstore.New(mc), postgres.NewStore(pool), map[string]port.Translator{
		"zip": applehealth.Translator{}, "fit": fit.Translator{}, "gpx": gpx.Translator{}, "tcx": tcx.Translator{},
	}, ingest.Config{
		MaxAttempts: envInt("INGEST_MAX_ATTEMPTS", ingest.DefaultMaxAttempts),
		BatchSize:   envInt("INGEST_COPY_BATCH", ingest.DefaultBatchSize),
	}, log)
	cons, err := consumer.New(conn, uc, log)
	if err != nil {
		return err
	}
	relay := outbox.NewRelay(pool, pub, time.Second, log)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { relay.Run(gctx); return nil })
	g.Go(func() error { return cons.Run(gctx) })
	g.Go(func() error {
		select { // a queda da conexão derruba o processo; o orquestrador reinicia
		case err := <-conn.NotifyClose(make(chan *amqp.Error, 1)):
			if err != nil {
				return err
			}
		case <-gctx.Done():
		}
		return nil
	})
	log.Info("worker iniciado")
	return g.Wait()
}

func need(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", errors.New("variável de ambiente obrigatória ausente: " + name)
	}
	return v, nil
}

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

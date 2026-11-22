// Comando worker: consome TreinoImportado e AtividadeCorrigida, calcula a carga e publica MetricasCalculadas.
// Configuração só por variáveis de ambiente; encerra em SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/activities"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/jwtauth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/outbox"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/postgres"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/adapters/rabbit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/consumer"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/httpapi"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/telemetry"
)

const defaultAddr = ":8121"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}
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
	activitiesURL, err := need("ATIVIDADES_BASE_URL")
	if err != nil {
		return err
	}
	serviceToken, err := need("INTERNAL_API_TOKEN")
	if err != nil {
		return err
	}
	keyFile, err := need("JWT_PUBLIC_KEY_FILE")
	if err != nil {
		return err
	}
	auth, err := jwtauth.FromFile(keyFile, os.Getenv("JWT_ISSUER"), envOr("JWT_AUDIENCE", "athlete-api"))
	if err != nil {
		return err
	}
	shutdown, err := telemetry.Setup(ctx, "metrics")
	if err != nil {
		return fmt.Errorf("telemetria: %w", err)
	}
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(sctx); err != nil {
			log.Warn("encerrar telemetria", "erro", err)
		}
	}()
	amqpURL, err := need("RABBITMQ_URL")
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, metricsURL)
	if err != nil {
		return fmt.Errorf("postgres metricas: %w", err)
	}
	defer pool.Close()

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

	svc := metrics.NewService(postgres.NewStore(pool), activities.NewClient(activitiesURL, serviceToken), time.Now)
	imported, err := consumer.New(conn, consumer.QueueImported, consumer.Imported(svc), log)
	if err != nil {
		return err
	}
	corrected, err := consumer.New(conn, consumer.QueueCorrected, consumer.Corrected(svc), log)
	if err != nil {
		return err
	}
	athletes := athlete.NewService(postgres.NewAthleteStore(pool), time.Now)
	links, err := consumer.New(conn, consumer.QueueLinks, consumer.Links(athletes), log)
	if err != nil {
		return err
	}
	relay := outbox.NewRelay(pool, pub, time.Second, log)
	srv := &http.Server{Addr: envOr("HTTP_ADDR", defaultAddr), Handler: httpapi.New(athletes, auth).Handler(), ReadHeaderTimeout: 10 * time.Second}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return imported.Run(ctx) })
	g.Go(func() error { return corrected.Run(ctx) })
	g.Go(func() error { return links.Run(ctx) })
	g.Go(func() error { relay.Run(ctx); return nil })
	g.Go(func() error {
		go func() {
			<-ctx.Done()
			sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(sctx)
		}()
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	log.Info("serviço de métricas iniciado")
	return g.Wait()
}

// healthcheck serve ao HEALTHCHECK do compose: a imagem distroless não tem shell nem curl.
func healthcheck() int {
	addr := envOr("HTTP_ADDR", defaultAddr)
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + addr + "/health")
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func need(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("variável %s não definida", name)
	}
	return v, nil
}

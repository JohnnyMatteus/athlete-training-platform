// Package ingest é o caso de uso do worker: lê o arquivo do lote, traduz, grava e publica eventos.
package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
)

const (
	DefaultMaxAttempts = 3
	DefaultBatchSize   = 5000
)

// Config ajusta o caso de uso.
type Config struct {
	MaxAttempts int
	BatchSize   int
}

// IngestImport processa uma ImportacaoRecebida.
type IngestImport struct {
	objects     port.ObjectStore
	uow         port.UnitOfWork
	translators map[string]port.Translator // por tipo_arquivo: zip, fit, tcx, gpx
	cfg         Config
	log         *slog.Logger
	now         func() time.Time
}

func New(objects port.ObjectStore, uow port.UnitOfWork, translators map[string]port.Translator, cfg Config, log *slog.Logger) *IngestImport {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultMaxAttempts
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if log == nil {
		log = slog.Default()
	}
	return &IngestImport{objects: objects, uow: uow, translators: translators, cfg: cfg, log: log, now: time.Now}
}

// Handle processa a mensagem. Erro devolvido = reentregar (nack com requeue);
// nil = confirmar (ack), inclusive quando a mensagem já foi processada ou o lote falhou em definitivo.
func (u *IngestImport) Handle(ctx context.Context, msg port.ImportReceived) error {
	err := u.process(ctx, msg)
	if err == nil {
		return nil
	}
	permanent := errors.Is(err, port.ErrMalformed) || errors.Is(err, port.ErrUnsupported)
	if !permanent && msg.Attempt < u.cfg.MaxAttempts {
		u.log.WarnContext(ctx, "falha na ingestão, será reentregue", "import_id", msg.ImportID, "tentativa", msg.Attempt, "erro", err)
		return err
	}
	if ferr := u.fail(ctx, msg, err); ferr != nil {
		return errors.Join(err, fmt.Errorf("registrar falha: %w", ferr))
	}
	u.log.ErrorContext(ctx, "lote falhou", "import_id", msg.ImportID, "tentativas", msg.Attempt, "erro", err)
	return nil
}

func (u *IngestImport) fail(ctx context.Context, msg port.ImportReceived, cause error) error {
	code := port.CodeInternal
	switch {
	case errors.Is(cause, port.ErrMalformed):
		code = port.CodeInvalidFile
	case errors.Is(cause, port.ErrUnsupported):
		code = port.CodeUnsupportedFormat
	}
	return u.uow.Do(ctx, func(ctx context.Context, tx port.Tx) error {
		if first, err := tx.MarkProcessed(ctx, msg.EventID); err != nil || !first {
			return err
		}
		return tx.Enqueue(ctx, port.Event{
			ID: uuid.New(), Type: port.EventImportFailed, RoutingKey: port.RouteImportFailed, ImportID: msg.ImportID,
			OccurredAt: u.now().UTC(),
			// o motivo é o texto do erro de domínio; não carrega dado de saúde
			Payload: port.ImportFailed{AthleteID: msg.AthleteID, Code: code, Reason: cause.Error(), Attempts: msg.Attempt},
		})
	})
}

func (u *IngestImport) process(ctx context.Context, msg port.ImportReceived) error {
	tr, ok := u.translators[msg.FileType]
	if !ok {
		return fmt.Errorf("%w: %q", port.ErrUnsupported, msg.FileType)
	}
	obj, err := u.objects.Open(ctx, msg.Bucket, msg.ObjectKey)
	if err != nil {
		return fmt.Errorf("abrir arquivo: %w", err)
	}
	defer func() { _ = obj.Close() }() // só leitura do objeto; falha ao fechar não afeta o resultado

	return u.uow.Do(ctx, func(ctx context.Context, tx port.Tx) error {
		first, err := tx.MarkProcessed(ctx, msg.EventID)
		if err != nil {
			return err
		}
		if !first {
			u.log.InfoContext(ctx, "mensagem já processada, ignorada", "event_id", msg.EventID)
			return nil
		}
		s := newSink(msg, tx, u.cfg.BatchSize, u.now)
		report, err := tr.Translate(ctx, msg.AthleteID, obj, s)
		if err != nil {
			return err
		}
		if err := s.flush(ctx); err != nil {
			return err
		}
		s.totals.Errors += report.Skipped
		return tx.Enqueue(ctx, s.finish(report.FormatVersion)...)
	})
}

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
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
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
	imports     port.ImportStore
	objects     port.ObjectStore
	uow         port.UnitOfWork
	translators map[activity.Source]port.Translator
	cfg         Config
	log         *slog.Logger
	now         func() time.Time
}

func New(imports port.ImportStore, objects port.ObjectStore, uow port.UnitOfWork,
	translators map[activity.Source]port.Translator, cfg Config, log *slog.Logger) *IngestImport {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultMaxAttempts
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if log == nil {
		log = slog.Default()
	}
	return &IngestImport{imports: imports, objects: objects, uow: uow, translators: translators, cfg: cfg, log: log, now: time.Now}
}

// Handle processa a mensagem. Erro devolvido = a mensagem deve ser reentregue (nack);
// nil = pode confirmar (ack), inclusive quando o lote já terminou ou falhou em definitivo.
func (u *IngestImport) Handle(ctx context.Context, msg port.ImportReceived) error {
	imp, started, err := u.imports.Begin(ctx, msg.ImportID)
	if err != nil {
		return fmt.Errorf("reservar lote: %w", err)
	}
	if !started {
		u.log.InfoContext(ctx, "lote já processado, mensagem ignorada", "import_id", msg.ImportID)
		return nil
	}

	err = u.process(ctx, imp)
	if err == nil {
		return nil
	}
	permanent := errors.Is(err, port.ErrMalformed)
	if !permanent && imp.Attempts < u.cfg.MaxAttempts {
		u.log.WarnContext(ctx, "falha na ingestão, será reentregue", "import_id", imp.ID, "tentativa", imp.Attempts, "erro", err)
		return err
	}
	if ferr := u.imports.Fail(ctx, imp.ID, err.Error(), imp.Attempts); ferr != nil {
		return errors.Join(err, fmt.Errorf("marcar lote como falho: %w", ferr))
	}
	u.log.ErrorContext(ctx, "lote falhou", "import_id", imp.ID, "tentativas", imp.Attempts, "erro", err)
	return nil
}

func (u *IngestImport) process(ctx context.Context, imp port.Import) error {
	tr, ok := u.translators[imp.Format]
	if !ok {
		return fmt.Errorf("%w: formato %q sem tradutor", port.ErrMalformed, imp.Format)
	}
	obj, err := u.objects.Open(ctx, imp.ObjectKey)
	if err != nil {
		return fmt.Errorf("abrir arquivo: %w", err)
	}
	defer obj.Close()

	return u.uow.Do(ctx, func(ctx context.Context, tx port.Tx) error {
		s := newSink(imp, tx, u.cfg.BatchSize, u.now)
		report, err := tr.Translate(ctx, imp.AthleteID, obj, s)
		if err != nil {
			return err
		}
		if err := s.flush(ctx); err != nil {
			return err
		}
		s.totals.Errors += report.Skipped
		s.events = append(s.events, port.Event{
			ID: uuid.New(), Type: port.EventImportDone, OccurredAt: u.now().UTC(),
			Payload: port.ImportDone{ImportID: imp.ID, Totals: s.totals},
		})
		if err := tx.CompleteImport(ctx, imp.ID, s.totals); err != nil {
			return err
		}
		return tx.Enqueue(ctx, s.events...)
	})
}

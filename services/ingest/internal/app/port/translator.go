// Package port define as portas da aplicação de ingestão. Só expõe tipos de domínio ou primitivos.
package port

import (
	"context"
	"errors"
	"io"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// ErrMalformed marca arquivo ilegível ou truncado: repetir o processamento não ajuda.
var ErrMalformed = errors.New("arquivo malformado")

// ErrUnsupported marca um tipo de arquivo sem tradutor.
var ErrUnsupported = errors.New("formato não suportado")

// Source é o arquivo a traduzir, com acesso aleatório (necessário para ler o zip sem carregá-lo).
type Source interface {
	io.ReaderAt
	Size() int64
}

// Sink recebe o que o tradutor produz, uma atividade e um lote de amostras por vez.
type Sink interface {
	// Activity entrega uma atividade completa (sem amostras). Devolve false se ela deve ser
	// descartada (duplicada); o tradutor então pode pular as amostras dela.
	Activity(ctx context.Context, a *activity.Activity) (accept bool, err error)
	// Samples entrega amostras de uma atividade já aceita. O lote só vale durante a chamada.
	Samples(ctx context.Context, id activity.ID, batch []activity.Sample) error
}

// Report resume uma tradução.
type Report struct {
	Activities    int
	Samples       int
	Skipped       int // registros descartados por valor inválido
	Warnings      []string
	FormatVersion string
}

// Translator é a porta de saída da ACL: converte um formato de terceiros em tipos de domínio.
type Translator interface {
	Translate(ctx context.Context, athlete AthleteID, src Source, sink Sink) (Report, error)
}

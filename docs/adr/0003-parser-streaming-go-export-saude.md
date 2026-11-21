# 0003. Parser em streaming, em Go, para o export do app Saúde (A-03)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-03 (seção 13; detalhe nas seções 3 e 5.1)

## Contexto

Não existe API web para dados do Apple Watch; uma das portas de entrada é o export do app Saúde, um .zip com `export.xml` de centenas de MB a GB, `export_cda.xml` e `workout-routes/*.gpx` (seção 3). Carregar o XML inteiro em memória não é viável. O SLO é processar 95% das importações de até 500 MB em menos de 5 min (seção 10).

## Decisão

A ingestão do export é feita por um worker em Go com streaming de ponta a ponta: `archive/zip` + `xml.Decoder.Token()`, memória constante independentemente do tamanho do export. O upload é resumível (tus) para o armazenamento de objetos. A escrita é em lote: `COPY` no Postgres para amostras e upsert de atividades. A idempotência usa a chave natural por atividade `hash(fonte, inicio_utc, duracao, tipo, dispositivo)`, de modo que reimportar o mesmo export não duplica nada.

## Alternativas consideradas

- Carregar o XML inteiro (DOM): descartado pelo próprio design doc ("nunca carregar o XML inteiro", seção 3).
- Demais alternativas (outra linguagem para o parser) não estão registradas; ver ADR 0007.

## Consequências

- Memória previsível, verificada pelo teste de carga com export sintético de 1 GB (seção 11).
- Reimportação idempotente, verificada por teste de propriedade.
- Parsing token a token é mais trabalhoso de escrever e testar do que um mapeamento declarativo.

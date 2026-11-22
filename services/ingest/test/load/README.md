# Teste de carga da ingestão (M1-14)

Gera exports sintéticos (tamanho de `export.xml` descompactado) e processa com o caso de uso do
worker. Não roda em `go test ./...` (tag `load`); o arquivo de 1 GB nunca entra no Git.

```bash
go test -tags=load -timeout 40m -v ./test/load          # 100, 500 e 1024 MB
LOAD_SIZES_MB=100,300 go test -tags=load -v ./test/load  # tamanhos à escolha
go run ./tools/synthetic-export -size-mb 1024 -seed 42 -out /tmp/export-1gb.zip   # só o arquivo
```

Critérios: pico de heap varia menos de 48 MB entre os tamanhos; 500 MB em menos de 5 min; mesma
semente gera o mesmo sha256. Sem `LOAD_DATABASE_URL` o destino é nulo (mede tradução,
deduplicação e lotes); a medição com Postgres real fica para o job do CI.

## Resultados

Registrar aqui a saída `RESULTADO ...` junto com a máquina usada (o design doc não define o
ambiente de referência do SLO).

| Data | Máquina | Tamanho | Pico de heap | Tempo |
| --- | --- | --- | --- | --- |
| (pendente: rodar no job manual do CI) | | | | |

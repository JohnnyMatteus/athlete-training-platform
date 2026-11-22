# services

| Diretório | Linguagem | Marco | Papel |
| --- | --- | --- | --- |
| `api` | PHP (Hyperf) | M1 | API HTTP, importação, outbox, hook do tusd; Planejamento no M4 |
| `ingest` | Go | M1 | Worker de ingestão: tradutores, `COPY`, eventos |
| `metrics` | Go | M2 | Métricas de carga |
| `coach` | TypeScript | M6 | Coach de IA |

## Convenção para todo serviço

- Estrutura hexagonal (design doc 4.3): `Domain/`, `Application/`, `Infrastructure/{Persistence,Messaging,Acl}/`, `Interfaces/`.
- O CI chama estes comandos; cada serviço os expõe:
  - PHP: `composer lint`, `composer test`, e opcionalmente `composer test:integration`.
  - Go: `go vet ./...`, `golangci-lint run`, `go test ./...` e `go test -tags=integration ./...`.
- Configuração só por variáveis de ambiente (nomes em `.env.example` na raiz).
- Eventos e HTTP seguem `contracts/`; mudança de contrato e de serviço andam no mesmo PR.
- Cada serviço é dono do próprio banco (`infra/db/migrations/<banco>/`) e nunca lê o de outro.

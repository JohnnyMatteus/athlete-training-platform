# ADRs: athlete-training-platform

Decisões iniciais do [design doc](../design-doc.md) (seção 13), no formato de `docs/adr/0000-template.md`. Todas com status **Proposto** até o autor aceitar. A numeração local (`NNNN`) segue a ordem dos ids do design doc (`A-NN`).

| ADR | Id | Decisão | Afeta o M1 |
| --- | --- | --- | --- |
| [0001](0001-contextos-delimitados-por-eventos.md) | A-01 | Seis contextos delimitados integrados por eventos (outbox + RabbitMQ) | sim |
| [0002](0002-acl-para-fontes-e-destinos-externos.md) | A-02 | ACL para toda fonte e destino externo; nenhum tipo de terceiro no domínio | sim |
| [0003](0003-parser-streaming-go-export-saude.md) | A-03 | Parser em streaming (Go) para o export do app Saúde | sim |
| [0004](0004-timescaledb-para-amostras.md) | A-04 | TimescaleDB para amostras; Postgres comum para o resto | sim |
| [0005](0005-motor-deterministico-ia-propoe.md) | A-05 | Motor de periodização determinístico; IA só propõe, o validador decide | não (M4, M6) |
| [0006](0006-plugin-capacitor-workoutkit.md) | A-06 | Plugin Capacitor próprio em Swift para o WorkoutKit, publicado como open source | não (M5) |
| [0007](0007-hyperf-api-planejamento-go-ingestao-metricas.md) | A-07 | Hyperf para API e Planejamento; Go para ingestão e métricas | sim |
| [0008](0008-dados-de-saude-sensiveis-lgpd.md) | A-08 | Dados de saúde tratados como sensíveis (LGPD): consentimento, criptografia, retenção curta de arquivos | sim (dados sintéticos, retenção) |
| [0009](0009-autenticacao-jwt-do-atleta.md) | n/a | Autenticação do atleta com JWT (RS256) validado pela API; provedor fora do M1 | sim |

Onde o design doc não registra alternativas, o ADR diz isso explicitamente em vez de inventar; o autor completa antes de aceitar.

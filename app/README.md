# app

App Ionic + Vue 3 + Capacitor 8 (identidade "Fôlego", `design/direcao-de-identidade.md` seção 13). TanStack Query, zod e Pinia. ADRs 0030 a 0034.

```bash
cd app
npm ci
npm run dev        # navegador; o login aceita um JWT colado só no build de desenvolvimento
npm run lint && npm run typecheck && npm test && npm run build
```

Uma URL por serviço em `.env.example`. O plugin nativo vive em `../plugins/capacitor-workoutkit` e é consumido por alias (nada é publicado no npm).

## Telas e o que as alimenta

| Tela | Fonte | Estado |
| --- | --- | --- |
| Hoje, Carga (CTL/ATL/TSB, fôlego) | `GET /metricas/carga` (8121) | contrato publicado; indisponível enquanto o serviço não responder |
| Atividades | `GET /metricas/atividades` | idem; mostra carga, não duração/distância (sem endpoint de leitura de atividades) |
| Limiares e perfil | `GET/PUT /metricas/limiares`, `/metricas/perfil` | idem |
| Zonas | nenhum endpoint | sempre indisponível |
| Plano, Calendário, aderência | `/planos`, `/planos/{id}`, `/planos/{id}/aderencia` (8120) | contrato publicado |
| Importar | `/imports` + tus (9501) | contrato publicado, consentimento por categoria, upload retomável |
| Coach | `/coach/messages` (proposto) | sem contrato: indisponível |
| Leitura de saúde | `/health/workouts` (proposto) | leitura e âncora prontas; entrega indisponível sem endpoint |

Nada foi exercitado contra os serviços reais nem em aparelho: os testes usam dublês e os YAML de `contracts/`.

## Não coberto

Criar plano (`POST /planos`), aprovar ajuste, vínculo com treinador, corrigir atividade (o cliente existe em `api/atividades.ts`, sem tela), apagar conta e exportar dados, leitura em background, E2E Playwright, projetos `ios/` e `android/` (gerados por `npx cap add`, ausentes por causa do disco).

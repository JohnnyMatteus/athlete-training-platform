# 0013. Serviço de Planejamento em Hyperf, banco `planejamento`, hospedando o vínculo atleta-treinador

- **Status:** Proposto
- **Data:** 2026-11-22

## Contexto

O ADR 0007 fixa Hyperf para API e Planejamento, e o design doc (4.2) desenha Planejamento como caixa própria. O contexto Atleta (perfil, metas, vínculo) ainda não tem serviço. A ADR 0009 pede um banco por contexto.

## Decisão

- Novo serviço `services/planning` (Hyperf, hexagonal), com banco `planejamento` (`infra/db/migrations/planejamento/`), porta de host 8120 e exchange `planejamento`.
- Mesmo desenho do serviço `api`: outbox transacional, inbox por `event_id`, JWT RS256 validado numa ACL (ADR 0011).
- O serviço hospeda, de forma provisória, o agregado `VinculoTreinador` do contexto Atleta (ADR 0014) e publica seus eventos numa exchange própria, `atleta`. Quando o contexto Atleta ganhar serviço, o agregado e a exchange migram sem mudar contrato.
- Planejamento não lê outro banco: CTL/ATL e TSS realizado vêm de `MetricasCalculadas`, limiares de `LimiarAtualizado`, ambos gravados em read models próprios.
- Perfil fisiológico e limiares continuam no banco `metricas` (ADR 0012); a API de escrita deles é do serviço de métricas, não de Planejamento (corrige a frase do ADR 0012 que apontava para a API de Planejamento).

## Alternativas consideradas

- Módulo dentro de `services/api`: menos infraestrutura, mas mistura importação com planejamento num mesmo deploy e num mesmo banco, contra a ADR 0009. Decisão do autor antes de aceitar.
- Serviço do contexto Atleta já separado: prematuro para um agregado só.

## Consequências

- Mais um serviço PHP para manter (CI, imagem, compose). Código de infraestrutura comum (JWT, outbox) fica duplicado entre `api` e `planning`; extrair um pacote Composer compartilhado é evolução.

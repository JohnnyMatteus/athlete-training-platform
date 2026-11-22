# 0024. Modo treinador no Coach respeita o vínculo consentido

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** ADR 0014

## Decisão

- Papel vem da claim `roles` (`atleta`, `treinador`, `admin`; ausente é atleta; quem tem os dois age como atleta; só `admin` é recusado, pois não lê dado de saúde).
- Treinador nomeia o atleta na requisição. O Coach só lê, e só propõe, se `GET /vinculos` mostrar vínculo `ativo` daquele treinador com aquele atleta e a categoria `treinos` (ler) ou `plano` (aprovar). A checagem é repetida no momento da aprovação, então revogação entre proposta e aprovação barra a aplicação.
- Os serviços donos continuam autorizando com o mesmo token; a checagem do Coach é defesa em profundidade e dá erro claro.
- Atleta pedindo dado de outro atleta recebe 403; proposta ou atleta desconhecido para quem não pode ver recebe 404, como na API.

## Consequências

- `frequencia_cardiaca` não é usada pelo Coach (FC não vai ao modelo).
- Consistência do vínculo depende do Planejamento.

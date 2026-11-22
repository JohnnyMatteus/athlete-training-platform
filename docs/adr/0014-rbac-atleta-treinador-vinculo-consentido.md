# 0014. RBAC atleta/treinador com vínculo consentido

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 9 ("treinador só vê atletas que o convidaram")

## Contexto

O ADR 0011 deixou os papéis para depois. O design doc exige RBAC (atleta, treinador vinculado, admin) e que o treinador só veja atletas que consentiram (LGPD, ADR 0008).

## Decisão

- Papéis na claim `roles` do JWT (`atleta`, `treinador`, `admin`); ausente equivale a `[atleta]`. `sub` continua sendo o usuário.
- Vínculo `VinculoTreinador` (atleta, treinador, categorias consentidas, status `pendente` → `ativo` → `revogado`): o atleta convida (consentimento), o treinador aceita, qualquer um dos dois revoga. Categorias: `treinos`, `frequencia_cardiaca`, `plano`.
- Treinador lê dados de um atleta só com vínculo `ativo`; sem vínculo a resposta é 404, igual a "não existe". Treinador nunca escreve dado de saúde do atleta (perfil, limiares, importação); só aprova e propõe ajustes ao plano.
- `admin` não lê dado de saúde; serve a operação (fora do escopo desta entrega, sem endpoint).
- Cada serviço decide localmente: Planejamento consulta o próprio banco; Métricas mantém um read model `vinculo_treinador` alimentado por `VinculoTreinadorConsentido` e `VinculoTreinadorRevogado` (nenhuma leitura entre bancos). Janela de consistência eventual: revogação vale quando o evento chega (normalmente ms); o treinador não ganha acesso novo sem o evento de consentimento.
- A API de importação (`services/api`) não ganha leitura de treinador: ninguém além do dono lê lotes.

## Alternativas consideradas

- Consulta síncrona do vínculo a cada requisição: consistência imediata, mas acopla os serviços e a disponibilidade de um ao outro.
- Claim do token listando atletas atendidos: revogação só no `exp`, inaceitável para consentimento.

## Consequências

- Revogação reflete com atraso de uma entrega de evento nos serviços consumidores.
- Provedor de identidade precisa emitir `roles`; no desenvolvimento o script de chaves gera tokens com o papel pedido.

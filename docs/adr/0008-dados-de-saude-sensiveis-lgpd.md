# 0008. Dados de saúde tratados como sensíveis (LGPD) (A-08)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-08 (seção 13; detalhe na seção 9)

## Contexto

Dados de saúde são dado pessoal sensível pela LGPD (seção 3). Vazamento é um risco listado (seção 15).

## Decisão

- Consentimento explícito e granular por tipo de dado, revogável.
- Criptografia em trânsito (TLS) e em repouso (disco e backups); segredos no Vault.
- Arquivos de importação apagados do armazenamento após processamento bem-sucedido (retenção configurável, padrão 7 dias).
- Exclusão real: "apagar minha conta" remove atividades, amostras, planos e arquivos; registro de auditoria sem dado de saúde.
- Autorização por atleta (RBAC: atleta, treinador vinculado, admin); treinador só vê atletas que o convidaram.
- Desenvolvimento e testes só com dados sintéticos ou anonimizados; dados reais do autor ficam locais e fora do Git.

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Arquivos de contrato da ACL precisam ser anonimizados antes de entrar no repositório.
- Exclusão real exige que todo contexto saiba apagar os dados de um atleta.
- Retenção curta impede reprocessar um export antigo sem novo upload.

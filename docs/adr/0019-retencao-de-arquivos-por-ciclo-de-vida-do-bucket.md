# 0019. Retenção de 7 dias dos arquivos de importação por regra de ciclo de vida do bucket

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 9 (retenção configurável, padrão 7 dias)

## Contexto

Arquivos de importação contêm dado de saúde e não podem ficar indefinidamente (ADR 0008). O design doc fixa retenção padrão de 7 dias.

## Decisão

- Regra de expiração no bucket de uploads, aplicada por `minio-init`: `mc ilm rule add --expire-days ${UPLOADS_RETENTION_DAYS:-7}`. Vale para qualquer objeto, processado com sucesso, falho ou abandonado; o armazenamento apaga sozinho, sem job nosso que possa falhar calado.
- Em S3 real, a mesma regra vira `LifecycleConfiguration` do bucket (documentada no README da infraestrutura).
- O worker não apaga o arquivo ao terminar: a janela de 7 dias permite reprocessar uma falha sem novo upload. Quem quiser retenção menor reduz a variável.

## Alternativas consideradas

- Varredura no worker: lógica própria e mais um ponto de falha.
- Apagar logo após sucesso: conflita com a janela de reprocesso do design doc.

## Consequências

- Reprocessar um lote com mais de 7 dias exige novo upload.

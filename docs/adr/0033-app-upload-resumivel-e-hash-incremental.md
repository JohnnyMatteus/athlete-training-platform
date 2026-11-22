# 0033. App: upload resumível por tus, hash incremental e retomada do lote

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 5.1; implementa o contrato do ADR 0010

## Decisão

- `tus-js-client` atrás da porta `ResumableUploader`, em fatias de 8 MiB, com o token lido a cada requisição.
- sha256 calculado em fatias (`@noble/hashes`), pois o export tem GBs. O `POST /imports` só ocorre depois do hash.
- Envio interrompido volta ao mesmo lote: o app guarda `import_id` e URL tus por (nome, tamanho, data) e confere `GET /imports/{id}` (`aguardando_upload`). Senão cria outro.
- O contrato não lista lotes; o app guarda os ids dos próprios envios e consulta cada um.
- Consentimento obrigatório antes de criar o lote; o payload sai do `ConsentRepository`.

## Consequências

- O texto jurídico do termo é do autor; o do app é provisório (versão `2026-11`).
- Ler arquivo gigante pela webview depende do seletor do sistema; sem teste em aparelho.

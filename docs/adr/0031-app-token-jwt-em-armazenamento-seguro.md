# 0031. App: JWT no Keychain/Keystore, nunca em localStorage

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** não existe; complementa a seção 9 e o ADR 0011

## Contexto

O JWT do atleta dá acesso a dado de saúde. `localStorage` e Preferences são legíveis por qualquer script na webview e entram em backups.

## Decisão

- No nativo, o token fica em `@aparajita/capacitor-secure-storage` (Keychain no iOS, Keystore no Android), atrás da porta `TokenStore`.
- Na web (desenvolvimento), o token fica só em memória e some ao recarregar.
- O cliente HTTP lê o token a cada requisição (inclusive o upload tus, via `onBeforeRequest`); 401 limpa a sessão.
- Login: sem provedor de identidade escolhido (ADR 0011), o build de produção mostra que o login não está disponível; só o build de desenvolvimento aceita um JWT colado. O fluxo real depende do provedor.

## Alternativas consideradas

- Preferences/localStorage: descartado pelo risco acima.
- Refresh token: depende do provedor; fora do escopo.

## Consequências

- Autor precisa escolher o provedor para fechar o login; o `TokenStore` não muda.
- O Keychain sobrevive à reinstalação no iOS por padrão; avaliar limpar no primeiro uso.

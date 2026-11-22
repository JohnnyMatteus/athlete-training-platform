# 0009. Autenticação do atleta com JWT validado pela API (A-09)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** não existe; o design doc (seção 9) pede RBAC por atleta mas não escolhe provedor de identidade nem formato de credencial

## Contexto

A API de importação precisa saber qual atleta está chamando (lote pertence a um atleta, consentimento LGPD é por atleta, ADR 0008). O README de issues lista a autenticação como ponto em aberto (item 2) e a M1-04 supõe um atleta fixo em desenvolvimento.

## Decisão

- A API **valida** JWT, não emite. O provedor de identidade (Keycloak, Auth0, Cognito ou outro) fica fora do M1; a API depende só do contrato do token.
- Assinatura assimétrica (RS256) verificada com a chave pública configurada (`JWT_PUBLIC_KEY_FILE`); o segredo de assinatura nunca chega à API.
- Claims obrigatórias: `iss` igual a `JWT_ISSUER`, `aud` contendo `JWT_AUDIENCE`, `exp` presente e `sub` em UUID, que vira o `atleta_id`.
- A validação vive numa ACL (`Infrastructure/Acl/JwtAuthenticator`) atrás da porta `Authenticator`; nenhuma classe da biblioteca de JWT atravessa a porta.
- Lote de outro atleta responde 404 (não revela existência).
- Upload tus: o hook `pre-create` do tusd recebe o cabeçalho `Authorization` (encaminhado) e só autoriza o atleta dono do lote. Os hooks do tusd usam, além disso, um segredo compartilhado na URL, pois só trafegam na rede interna.
- Em desenvolvimento, o token é gerado por um script local com um par de chaves de teste; não há atleta fixo no código.
- Papéis (atleta, treinador, admin) entram quando houver contexto Atleta; a claim `roles` não é lida no M1.

## Alternativas consideradas

- O design doc não registra alternativas. Possíveis: sessão com cookie (ruim para app Capacitor), token opaco com introspecção (mais uma ida ao provedor por requisição), HS256 com segredo compartilhado (qualquer serviço que valida também poderia emitir). Decisão do autor antes de aceitar.

## Consequências

- Sem provedor escolhido, o app e os testes ponta a ponta usam um emissor de teste com o mesmo contrato.
- Rotação de chave exige reconfigurar a chave pública (JWKS fica como evolução).
- Revogação de token só pelo `exp` curto.

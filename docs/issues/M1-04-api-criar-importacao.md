# M1-04 · API: `POST /imports` e contrato OpenAPI

- **core:** false
- **curso:** fc4-apis-design
- **subagente:** implementador (Hyperf)
- **depende de:** M1-02
- **estimativa:** 1 dia

## Objetivo

Expor na API Hyperf o caso de uso "criar importação": recebe metadados do arquivo (nome, tamanho, tipo, sha256), cria o `LoteDeImportacao` e devolve `import_id` e `upload_url` (tus), conforme o diagrama da seção 5.1. Contrato descrito em OpenAPI.

## Critérios de aceite

```gherkin
Funcionalidade: Criar importação

  Cenário: Metadados válidos
    Dado metadados de um export.zip com sha256 de 64 caracteres hexadecimais
    Quando o app chama POST /imports
    Então a resposta é 201 com import_id e upload_url
    E existe um lote com status "aguardando_upload"

  Cenário: Mesmo arquivo enviado de novo
    Dado um lote concluído com sha256 S para o mesmo atleta
    Quando o app chama POST /imports com sha256 S
    Então a API cria um novo lote (a idempotência é por atividade, não por arquivo)

  Cenário: Metadados inválidos
    Dado um pedido sem sha256 ou com tipo de arquivo fora de zip, fit, tcx e gpx
    Quando o app chama POST /imports
    Então a resposta é 422 com o erro no formato definido no OpenAPI

  Cenário: Contrato publicado
    Dado o serviço em execução
    Quando acesso a especificação OpenAPI
    Então POST /imports e GET /imports/{id} estão descritos com exemplos
```

## Arquivos prováveis

- `services/api/src/Application/Import/CreateImport.php`
- `services/api/src/Domain/Import/LoteDeImportacao.php`
- `services/api/src/Interfaces/Http/ImportController.php`
- `services/api/openapi.yaml`

## Notas

- `GET /imports/{id}` (status e totais) entra aqui porque a tela de importação do M3 vai precisar; é leitura simples do lote.
- Autenticação e RBAC (seção 9) não estão no M1 do design doc. Esta issue supõe um atleta sintético fixo vindo de configuração no ambiente de desenvolvimento (ponto em aberto no README).
- O segundo cenário é uma leitura do design doc (idempotência por chave natural, seção 5.1); o autor pode preferir recusar sha256 repetido.

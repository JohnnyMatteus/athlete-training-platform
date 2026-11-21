# M1-05 · Upload resumível (tus) para o MinIO e hook de conclusão

- **core:** false
- **curso:** fc4-sec-owasp-vault
- **subagente:** implementador + devops
- **depende de:** M1-04, M1-06
- **estimativa:** 1 dia

## Objetivo

Receber o arquivo por upload resumível (protocolo tus) gravando no MinIO e, quando o upload termina, chamar a API (hook) para conferir o sha256, marcar o lote como recebido e publicar `ImportacaoRecebida{import_id}` pela outbox.

## Critérios de aceite

```gherkin
Funcionalidade: Upload resumível

  Cenário: Upload interrompido e retomado
    Dado um arquivo sintético de 200 MB e um upload interrompido em 40%
    Quando o cliente retoma com o mesmo upload_url
    Então só os bytes restantes são enviados e o objeto final no MinIO tem o sha256 declarado

  Cenário: Hook de conclusão
    Dado um upload concluído
    Quando o hook chega à API
    Então o lote passa a "recebido" e um ImportacaoRecebida{import_id} é gravado na outbox

  Cenário: sha256 divergente
    Dado um upload cujo conteúdo não confere com o sha256 do lote
    Quando o hook chega à API
    Então o lote passa a "falhou" com motivo, nenhum ImportacaoRecebida é publicado e ImportacaoFalhou é publicado

  Cenário: upload_url de outro lote
    Dado um upload_url emitido para o lote A
    Quando alguém tenta usá-lo depois que o lote A foi concluído ou expirou
    Então o servidor tus recusa
```

## Arquivos prováveis

- `deploy/compose/docker-compose.yml` (servidor tus com backend S3/MinIO)
- `services/api/src/Interfaces/Http/UploadHookController.php`
- `services/api/src/Application/Import/ConfirmUpload.php`

## Notas

- O design doc diz "Upload resumível (tus)" sem escolher a implementação do servidor tus (ex.: tusd com backend S3, ou servidor tus dentro do Hyperf). Esta issue supõe um servidor tus separado com backend S3; decisão do autor.
- A retenção de 7 dias dos arquivos (ADR 0008) fica fora do M1; ver README.

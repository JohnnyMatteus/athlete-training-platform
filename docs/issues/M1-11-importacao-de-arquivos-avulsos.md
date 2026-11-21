# M1-11 · Importação de arquivos .fit, .gpx e .tcx avulsos

- **core:** false
- **curso:** fc4-apis-design
- **subagente:** implementador
- **depende de:** M1-05, M1-08, M1-09, M1-10
- **estimativa:** ½ dia

## Objetivo

Usar o mesmo fluxo de importação (POST /imports, tus, ImportacaoRecebida, worker) para arquivos .fit, .gpx e .tcx enviados um a um: o lote registra o tipo de arquivo e o worker escolhe o tradutor certo.

## Critérios de aceite

```gherkin
Funcionalidade: Arquivos avulsos

  Esquema do Cenário: Roteamento por tipo
    Dado um lote do tipo <tipo> com um arquivo válido
    Quando o worker processa
    Então o tradutor <tradutor> é usado e a atividade fica com fonte "<fonte>"

    Exemplos:
      | tipo | tradutor                    | fonte        |
      | zip  | AppleHealthExportTranslator | apple_export |
      | fit  | FitTranslator               | fit          |
      | gpx  | GpxTranslator               | gpx          |
      | tcx  | TcxTranslator               | tcx          |

  Cenário: Extensão e conteúdo divergentes
    Dado um lote do tipo fit cujo conteúdo é XML
    Quando o worker processa
    Então o lote falha com motivo "conteúdo não corresponde ao tipo"
```

## Arquivos prováveis

- `services/ingestion/src/Application/ingest/translator_registry.go`
- `services/api/openapi.yaml` (enum de tipo de arquivo)

## Notas

- O design doc só descreve o fluxo do export.zip; usar o mesmo fluxo para os outros formatos é uma suposição desta quebra.

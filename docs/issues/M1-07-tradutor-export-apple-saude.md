# M1-07 · `AppleHealthExportTranslator` em streaming

- **core:** false
- **curso:** fc4-arch-hexagonal
- **subagente:** implementador (Go) + testador
- **depende de:** M1-03; ADRs 0002, 0003
- **estimativa:** 1 dia

## Objetivo

Implementar o tradutor da ACL que lê o `export.zip` em streaming (`archive/zip` + `xml.Decoder.Token()`) e emite, um a um, `Atividade` e `Amostra` do domínio a partir dos elementos `Workout`, `Record` e `WorkoutRoute`, com unidades convertidas na fronteira e memória constante.

## Critérios de aceite

```gherkin
Funcionalidade: Tradução do export do app Saúde

  Cenário: Contrato com export reduzido anonimizado
    Dado testdata/apple/export-reduzido.zip com uma corrida, um pedal com potência e uma natação em piscina
    Quando o tradutor processa o arquivo
    Então emite 3 atividades com tipos Corrida, Ciclismo(outdoor) e Natacao(piscina)
    E as amostras de FC, potência, cadência e velocidade estão em bpm, W, rpm e m/s, com instantes em UTC

  Cenário: Tipo desconhecido
    Dado um Workout com workoutActivityType desconhecido
    Quando o tradutor processa
    Então a atividade é Outro e o tipo original fica nos metadados

  Cenário: Memória constante
    Dado um export sintético de 300 MB
    Quando o tradutor processa
    Então o pico de heap fica abaixo de um limite fixo definido no teste, independente do tamanho do arquivo

  Cenário: Nenhum tipo de terceiro na porta
    Dado a interface de saída do tradutor
    Então ela só expõe tipos de Domain/

  Cenário: XML malformado no meio do arquivo
    Dado um export truncado
    Quando o tradutor processa
    Então devolve as atividades completas lidas até o ponto do erro e um erro explícito com a posição
```

## Arquivos prováveis

- `services/ingestion/src/Infrastructure/Acl/applehealth/translator.go`
- `services/ingestion/src/Infrastructure/Acl/applehealth/translator_test.go`
- `services/ingestion/testdata/apple/` (export reduzido e anonimizado)

## Notas

- `export_cda.xml` e as rotas em `workout-routes/*.gpx` não têm uso definido no M1 pelo design doc; esta issue lê as rotas só se o GpxTranslator (M1-10) já existir, senão as ignora com aviso.
- O comportamento em XML truncado (último cenário) é uma proposta; o design doc não define.
- Registrar a versão do formato do export no lote (seção 15).

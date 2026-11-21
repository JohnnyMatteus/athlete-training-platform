# M1-09 · `FitTranslator`

- **core:** false
- **curso:** fc4-arch-hexagonal
- **subagente:** implementador (Go) + testador
- **depende de:** M1-03; ADR 0002
- **estimativa:** 1 dia

## Objetivo

Traduzir arquivos .fit de atividade (Garmin SDK; Wahoo e outros) em `Atividade`, `Volta` e `Amostra`, com unidades convertidas na fronteira e testes de contrato com arquivos reais anonimizados.

## Critérios de aceite

```gherkin
Funcionalidade: Tradução de .fit

  Cenário: Pedal com potência
    Dado testdata/fit/bike-potencia.fit anonimizado
    Quando o tradutor processa
    Então emite uma atividade Ciclismo com voltas e amostras de potência (W), cadência, FC (bpm) e velocidade (m/s)

  Cenário: Natação em piscina
    Dado testdata/fit/natacao-piscina.fit anonimizado
    Quando o tradutor processa
    Então a atividade é Natacao(piscina) e o comprimento da piscina fica nos metadados

  Cenário: Arquivo de outro fabricante
    Dado um .fit gravado por um ciclocomputador de outro fabricante
    Quando o tradutor processa
    Então a atividade é traduzida sem campos específicos do fabricante no domínio

  Cenário: Arquivo inválido
    Dado um arquivo com CRC inválido
    Quando o tradutor processa
    Então devolve erro explícito e nenhuma atividade parcial
```

## Arquivos prováveis

- `services/ingestion/src/Infrastructure/Acl/fit/translator.go`
- `services/ingestion/src/Infrastructure/Acl/fit/translator_test.go`
- `services/ingestion/testdata/fit/`

## Notas

- O design doc cita "Garmin SDK" sem definir a biblioteca Go; a escolha e a licença dela ficam para o autor.
- Coordenadas de GPS nos arquivos de teste precisam ser anonimizadas (ADR 0008).

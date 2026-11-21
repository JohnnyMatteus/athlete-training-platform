# M1-03 · Modelo de domínio de Atividades

- **core:** false
- **curso:** fc4-ddd-events
- **subagente:** implementador (Go)
- **depende de:** M1-01; ADR 0002
- **estimativa:** 1 dia

## Objetivo

Implementar em `Domain/` do worker o modelo canônico de treino realizado: agregado `Atividade` com `Sessao`, `Volta` e `Amostra`, o enum de tipo de atividade, value objects com unidades de domínio e o cálculo da chave natural, sem dependência de framework nem de biblioteca de parsing.

## Critérios de aceite

```gherkin
Funcionalidade: Modelo canônico de atividade

  Cenário: Enum de tipos
    Dado os tipos Natacao(piscina|aguas_abertas), Ciclismo(indoor|outdoor), Corrida, Forca e Outro
    Quando crio uma Atividade de tipo desconhecido "paddle"
    Então o tipo é Outro e o original "paddle" fica nos metadados

  Cenário: Unidades de domínio
    Dado uma amostra criada com velocidade em m/s, potência em W, FC em bpm e tempo em UTC
    Então os value objects recusam valores negativos e instantes sem fuso

  Cenário: Chave natural determinística
    Dado duas atividades com mesma fonte, inicio_utc, duração, tipo e dispositivo
    Quando calculo a chave natural
    Então as duas chaves são iguais
    E mudar qualquer um desses campos muda a chave

  Cenário: Evento de domínio
    Dado uma Atividade nova registrada
    Então o agregado emite AtividadeRegistrada
```

## Arquivos prováveis

- `services/ingestion/src/Domain/activity/*.go`
- `services/ingestion/src/Domain/activity/*_test.go`

## Notas

- Chave natural = `hash(fonte, inicio_utc, duracao, tipo, dispositivo)` (seção 5.1); o algoritmo de hash não está definido no design doc.
- As regras de qualidade de dado da seção 6 (descartar FC fora de 30 a 230 bpm e potência acima de 2.500 W) ficam para o M2; ver pontos em aberto no README.

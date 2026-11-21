# M1-14 · Teste de carga com export sintético de 1 GB

- **core:** false
- **curso:** fc4-tests-advanced
- **subagente:** testador
- **depende de:** M1-08
- **estimativa:** 1 dia

## Objetivo

Gerar um export do app Saúde sintético de 1 GB e provar que o worker processa com memória constante e tempo dentro do SLO da seção 10 (95% das importações de até 500 MB em menos de 5 min).

## Critérios de aceite

```gherkin
Funcionalidade: Carga da ingestão

  Cenário: Memória constante
    Dado exports sintéticos de 100 MB, 500 MB e 1 GB
    Quando o worker processa cada um
    Então o pico de memória varia menos que uma margem fixa definida no teste entre os três

  Cenário: Tempo do SLO
    Dado o export sintético de 500 MB
    Quando o worker processa no ambiente de referência
    Então termina em menos de 5 minutos

  Cenário: Gerador reproduzível
    Dado a mesma semente
    Quando o gerador cria o export
    Então o sha256 do arquivo é o mesmo
```

## Arquivos prováveis

- `tools/synthetic-export/` (gerador)
- `services/ingestion/test/load/`
- `docs/` (resultado da medição e ambiente de referência)

## Notas

- O arquivo de 1 GB não entra no Git; é gerado no job.
- O ambiente de referência para o SLO não está definido no design doc; registrar a máquina usada junto com o resultado.

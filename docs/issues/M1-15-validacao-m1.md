# M1-15 · Validação do critério de pronto do M1

- **core:** false
- **curso:** —
- **subagente:** testador
- **depende de:** M1-13, M1-14
- **estimativa:** ½ dia

## Objetivo

Conferir, num ambiente limpo, que o M1 (Ingestão) do plano de entrega está pronto: upload resumível, parser streaming do export.zip, tradutores FIT/GPX/TCX e deduplicação, e registrar as evidências no README das issues.

## Critérios de aceite

```gherkin
Funcionalidade: Critério de pronto do M1

  Cenário: Checklist do plano de entrega
    Dado o repositório na branch principal
    Quando sigo o README do zero (compose, migrações, serviços)
    Então consigo importar um export.zip sintético, um .fit, um .gpx e um .tcx anonimizados pela API
    E a reimportação não duplica atividades
    E o mesmo treino de duas fontes vira uma atividade principal e uma FonteAlternativa

  Cenário: Evidências
    Dado os testes de integração (M1-13) e de carga (M1-14)
    Então os dois estão verdes no CI e o resultado da medição está registrado
```

## Arquivos prováveis

- `docs/projects/athlete-training-platform/issues/README.md` (seção de validação)

## Notas

- O design doc não tem critério de pronto explícito por marco; o checklist acima deriva da linha do M1 no plano de entrega e precisa da aprovação do autor.

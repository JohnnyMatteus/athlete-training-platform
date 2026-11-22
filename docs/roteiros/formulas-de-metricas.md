# Roteiro: fórmulas de métricas (`formulas-de-metricas`)

Código: `services/metrics/internal/domain/load/`. Design doc, seção 6. Domínio puro: sem I/O, sem framework, testável com valores calculados à mão.

## A solução em uma página

1. **Qualidade antes da conta.** `Sanitize` remove FC fora de 30-230 bpm e potência acima de 2.500 W e devolve a contagem de descartes (`Discards`). A amostra continua; só o canal inválido some. O número vai para `carga_atividade` e para o evento `MetricasCalculadas`, então dá para auditar atividades com sensor ruim.
2. **Normalização comum.** NP, velocidade normalizada de natação e pace normalizado de corrida usam o mesmo núcleo (`normalizedMean`): série a 1 Hz, média móvel de 30 s, média da quarta potência, raiz quarta. Lacunas de até 30 s são interpoladas linearmente; maiores quebram a série em segmentos, e janelas nunca atravessam a quebra. Segmentos com menos de 30 s não formam janela.
3. **Escolha da fórmula** (`Calculate`): bike com potência e FTP, TSS por potência; corrida com pace de limiar, rTSS; natação com CSS, sTSS; senão, com FC e perfil (repouso, máxima, sexo), TRIMP; senão, carga zero com `MethodNone` (o registro existe, a carga não é inventada).
4. **Limiar versionado.** `ThresholdHistory.At(dia)` devolve, por espécie, a versão mais recente com `ValidFrom <= dia`. O TSS de uma atividade usa o limiar do dia dela, e uma mudança de FTP não reescreve o passado, a não ser que alguém peça o recálculo.
5. **Carga acumulada.** `Recompute(estado, from, until, tss)` aplica o filtro exponencial (42 e 7 dias) dia a dia, com TSS 0 nos dias sem treino, e `TSB(t) = CTL(t-1) - ATL(t-1)`. É função pura: mesma entrada, mesma saída. Corrigir uma atividade é recalcular a partir do dia dela com o estado do dia anterior.
6. **Aderência.** Razão realizado/planejado por sessão e por semana ISO. Sessão não planejada soma no realizado da semana; semana sem planejado fica de fora (sem razão definida).

## Decisões e alternativas

| Decisão | Alternativa | Por quê |
| --- | --- | --- |
| Interpolar até 30 s e quebrar acima | Preencher com zero; descartar o treino | Zero derruba o NP em pausa de sensor; descartar perde o treino. Quebrar não inventa dado. |
| Pace pelo custo de Minetti (2002) | Fator linear por % de subida; GAP do Strava | Polinômio publicado, simétrico para descida e fácil de conferir à mão (10% de subida = 1,65784). Inclinação limitada a ±30% contra ruído do altímetro. |
| rTSS = horas × IF² × 100 | Fórmula própria | É a mesma forma do TSS (`s × NP × IF / (FTP × 3600) × 100`), com velocidade no lugar de potência. |
| sTSS com IF³ | IF² | Está no design doc: a intensidade na água custa mais que linearmente (arrasto). |
| TRIMP gravado como carga sem reescala | Converter para escala de TSS | Escala de TRIMP não é a de TSS (1 h no limiar dá cerca de 100 de TSS e mais de 150 de TRIMP). O método vai junto (`metodo`) e a conversão fica como evolução. |
| Dia civil em UTC | Fuso do atleta | O contrato traz UTC. Treino de madrugada pode cair no dia seguinte; fuso por atleta é evolução. |
| Recálculo a partir do dia, com semente do dia anterior | Recalcular tudo | Custo proporcional ao que mudou. A propriedade testada garante o mesmo resultado. |

## Como foi testado

- Referências calculadas à mão em `formulas_test.go`: NP de 31 pontos (105,3555), NP com lacuna de 31 s (253,0440, que mudaria se houvesse interpolação), IF/TSS (96), sTSS (72,9), CSS (1,1111 m/s), TRIMP masculino e feminino (50,1446 e 59,4642), CTL/ATL do dia 0 e 1 (2,380952 e 14,285714), fator de Minetti, limites da faixa de descarte (29/30 e 230/231 bpm, 2.500/2.501 W).
- Testes de propriedade com `rapid` (`fitness_property_test.go`): recalcular do dia k com a semente do dia k-1 reproduz exatamente a série completa; repetir o recálculo dá o mesmo resultado; corrigir o dia k não muda os dias anteriores; CTL/ATL nunca negativos e `TSB = CTL(t-1) - ATL(t-1)`.
- `calculator_test.go`: seleção de método, limiar vigente na data, descartes contados e ignorados, queda para TRIMP quando só há FC.

## Perguntas de entrevista

1. Por que a raiz quarta da média da quarta potência, e não a média simples? (Esforço variável custa metabolicamente mais que o constante na mesma média; a quarta potência pesa os picos.)
2. O que acontece com NP se interpolar uma pausa de 5 minutos? E por que 30 s?
3. Como você garante que recalcular CTL/ATL é idempotente? Qual o risco se o recálculo lesse o estado já gravado dos dias afetados?
4. TSB usa o dia anterior. O que muda na interpretação de "forma" se usar o mesmo dia?
5. Limiar versionado: um FTP novo deve reescrever o TSS do passado? Quem decide e como o evento de recálculo entra?
6. Por que TRIMP e TSS não se somam sem cuidado? Como você reescalaria?
7. Que erros de dado derrubariam as métricas (potência de 65.535 W, FC 0) e onde você os barra?
8. Mesma atividade chegando duas vezes (reentrega do RabbitMQ): onde está a idempotência? (Inbox por `event_id` na mesma transação do efeito; ver integração.)

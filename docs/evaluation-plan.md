# mnemo — plan de evaluación

Estado: protocolo registrado, sin resultados · Última revisión: 2026-11-19

Este documento se escribe **antes** de medir. Lo que aquí se fija no se cambia después de ver resultados; si hay que cambiarlo, se anota en el registro de cambios del final, con fecha y motivo, y se reejecutan todos los brazos. Decisión de fondo: [ADR-0004](adr/ADR-0004-benchmark-titular-y-protocolo.md).

## 1. Preguntas que la evaluación responde

1. ¿mnemo responde mejor que un top-k vectorial y que el contexto completo, con el mismo lector?
2. ¿A qué coste en tokens, dólares y latencia por turno?
3. ¿Qué pieza aporta qué? (ablaciones)
4. ¿Dónde falla y por qué? (análisis de fallos)
5. ¿Resiste recuerdos inyectados y aísla inquilinos? (suites propias)

## 2. Conjuntos de datos

| Conjunto | Uso | Tamaño | Origen | Licencia | Notas |
|---|---|---|---|---|---|
| **LongMemEval_S** (limpio) | Titular | 500 preguntas; ~115.000 tokens y ~40 sesiones de historial por pregunta; 30 de abstención | `xiaowu0162/longmemeval-cleaned` en Hugging Face ([repositorio](https://github.com/xiaowu0162/LongMemEval)) | Repositorio MIT; licencia de los datos no declarada aparte (Q-5) | Tipos: `single-session-user`, `single-session-assistant`, `single-session-preference`, `temporal-reasoning`, `knowledge-update`, `multi-session`; abstención con sufijo `_abs` |
| **LongMemEval oráculo** | Diagnóstico | 500 preguntas con solo las sesiones de evidencia | mismo | mismo | Separa errores de recuperación de errores de lectura |
| **LoCoMo** (`locomo10.json`) | Secundario | 10 conversaciones; categorías 1-4 = 1.540 preguntas | [snap-research/locomo](https://github.com/snap-research/locomo) | CC BY-NC 4.0 | 99 errores conocidos en la clave ([auditoría](https://github.com/dial481/locomo-audit)); se reporta con y sin las preguntas señaladas |
| **Suite de envenenamiento** | Propia | 30 conversaciones con un recuerdo inyectado cada una | Escrita y etiquetada por mí | MIT (este repositorio) | No es un benchmark público; se presenta como prueba de regresión |
| **Suite temporal mínima** | Propia | 20 casos de hechos que cambian, con consulta actual y consulta con fecha | Escrita y etiquetada por mí | MIT | Prueba de aceptación de FR-6 y FR-8, no cifra titular |

Sin verificar: el recuento por tipo de pregunta de LongMemEval (solo aparece en una figura del artículo) y el número total de preguntas de LoCoMo incluida la categoría 5. El cargador los cuenta y los escribe en el informe.

Ningún conjunto se guarda en el repositorio. `mnemo-eval fetch` los descarga a `eval/data/` y comprueba su suma SHA-256 contra `eval/config/protocol.yaml`.

**División.** 60 preguntas de LongMemEval_S, estratificadas por tipo y elegidas con semilla fija, forman el **subconjunto de desarrollo**: es el único sobre el que se ajustan prompts y umbrales. Las otras 440 quedan **retenidas** y solo se ejecutan en la corrida final. Se publican tres cifras: las 500 (comparable con terceros), las 440 (sin contaminación de ajuste) y las 60.

## 3. Brazos

Todos comparten lector, prompt de lectura, juez y, salvo B0 y B1, presupuesto de contexto (8.000 tokens como máximo).

| Brazo | Qué es | Para qué |
|---|---|---|
| **B0 — sin memoria** | El lector recibe solo la pregunta y su fecha | Suelo: cuánto se acierta sin información |
| **B1 — contexto completo** | Todo el historial en el prompt, en orden cronológico | La alternativa "sin sistema de memoria"; también el coste a batir |
| **B2 — top-k vectorial** | Rondas embebidas con el mismo modelo, top-k por coseno hasta llenar el presupuesto | La línea base a superar; es mnemo con todo apagado salvo `episodes.vector` |
| **M — mnemo completo** | Configuración por defecto | Titular |

Ablaciones, cada una frente a M:

| ID | Cambio | Tipo | Hipótesis |
|---|---|---|---|
| A1 | Solo `episodes.fts` (léxico puro) | lectura | Se acerca a B2 más de lo esperado; un trabajo de 2026 reporta que la búsqueda léxica supera a la vectorial en LongMemEval dentro de un bucle de agente ([resumen de terceros](https://dev.to/pueding/is-grep-all-you-need-grep-vs-vector-retrieval-for-agentic-search-534k), artículo no leído) |
| A2 | Episodios vector + léxico, sin hechos | lectura | Aporta sobre B2 en preguntas con nombres propios |
| A3 | M sin fuentes de hechos | lectura | Mide el valor de la memoria semántica |
| A4 | M sin filtro de validez ni `as_of` (los hechos sustituidos compiten como activos) | lectura | Empeora `knowledge-update` y `temporal-reasoning` |
| A5 | M sin `facts.graph` | lectura | Empeora `multi-session` |
| A6 | M con reranker | lectura | Mejora la exactitud a cambio de latencia |
| A7 | M con consolidación tras cada 10 sesiones | escritura | Menos hechos activos sin perder exactitud |
| A8 | M con olvido a tres niveles de umbral | escritura | Curva exactitud frente a tamaño de la memoria |
| A9 | Presupuesto de contexto 2.000 / 4.000 / 8.000 tokens | lectura | Curva exactitud frente a tokens |

Las ablaciones de **lectura** se ejecutan sobre una única ingesta: solo cambian parámetros de `recall`. Las de **escritura** (A7, A8) necesitan reingestar y se ejecutan sobre 100 preguntas estratificadas, no sobre las 500. Esta separación es lo que hace que el plan quepa en el presupuesto.

Cada pregunta del benchmark es un sujeto distinto dentro de un inquilino creado para la corrida, así que el aislamiento del producto es también el aislamiento del experimento.

**Opcional (US-10):** un sistema de terceros de código abierto ejecutado por el mismo harness con el mismo lector y juez. Solo si queda presupuesto tras la corrida final.

## 4. Protocolo

1. **Ingesta.** Por pregunta: las sesiones del historial se envían en orden cronológico, una petición por sesión con su fecha, con `wait=true`. La extracción usa una caché por hash SHA-256 del contenido de la sesión y del identificador del prompt, de modo que repetir una corrida no vuelve a pagar.
2. **Consulta.** `recall` con `as_of` igual a la fecha de la pregunta, `track_access=false`, el presupuesto del brazo.
3. **Lectura.** Un único prompt para todos los brazos: instrucciones fijas, el bloque de memoria, la fecha actual, la pregunta, y la indicación de responder "no lo sé" si la memoria no contiene la respuesta. Temperatura 0.
4. **Juicio.** Los prompts oficiales por tipo de pregunta del script `evaluate_qa.py` de LongMemEval, con `gpt-4o-2024-08-06`, que es el juez del artículo (más del 97 % de acuerdo con expertos humanos, [artículo](https://arxiv.org/html/2410.10813v2)). Para LoCoMo, el mismo modelo juez; no se usa `gpt-4o-mini`, que la auditoría mostró laxo.
5. **Repeticiones.** Una corrida por brazo sobre las 500; el brazo M y B2 se repiten tres veces sobre las 60 de desarrollo para estimar la variación entre corridas. Se reporta el intervalo de confianza al 95 % por bootstrap sobre preguntas.
6. **Registro.** Por pregunta se guarda una traza: sesiones ingeridas, hechos extraídos, transiciones aplicadas, candidatos por fuente con su rango, contexto final, respuesta, veredicto del juez, tokens y tiempos. Las trazas son la materia del análisis de fallos.

Configuración fijada en `eval/config/protocol.yaml`: identificadores de modelo, parámetros, prompts por referencia a archivo, semilla, sumas de los datos y tabla de precios con fecha. El identificador del protocolo es el hash de ese archivo y aparece en cada tabla.

Modelos por defecto (supuesto A-3; son configuración):

| Papel | Modelo | Precio al 2026-11-19 | Fuente |
|---|---|---|---|
| Extractor y veredictos | Gemini 3.1 Flash-Lite | USD 0,25 / 1,50 por millón (entrada / salida); lotes a mitad | [precios](https://aistudio.google.com/docs/pricing) |
| Lector | Gemini 3.1 Flash-Lite | igual | igual |
| Embeddings | Qwen3-Embedding-0.6B, 1.024 dimensiones | local: 0; alojado: USD 0,012 por millón | [precios](https://developers.cloudflare.com/workers-ai/platform/pricing/) |
| Reranker (A6) | `bge-reranker-base` | USD 0,003 por millón | igual |
| Juez | `gpt-4o-2024-08-06` | sin verificar (Q-2) | — |

Los precios de Gemini Flash cambian el 2027-01-01 según su página; la tabla del protocolo lleva fecha y el informe la imprime.

## 5. Métricas

**Calidad**

- **Exactitud** (titular): fracción de preguntas que el juez da por correctas. Global y por tipo.
- **Exactitud de abstención**: sobre las 30 preguntas `_abs`. Contramétrica SM-C1.
- **Recall de recuperación**: recall@5 y recall@10 a nivel de sesión y de turno contra las sesiones de evidencia que trae el conjunto; se excluyen las 30 de abstención, como hace el script oficial. Se reporta por separado de la exactitud: una recuperación correcta con respuesta incorrecta es un fallo del lector, no de mnemo.
- **MRR** de la primera sesión de evidencia.

**Coste**

- **Tokens de entrada del lector por pregunta** (media y p95), tomadas del `usage` del proveedor.
- **Tokens de ingesta por sesión**: entrada y salida del extractor y de los veredictos, y tokens de embeddings.
- **Coste por turno** en USD: `(coste de ingesta de la sesión / turnos de la sesión) + coste de lectura de un turno`, donde la lectura incluye embeber la consulta, el reranker si aplica y las tokens de memoria añadidas al prompt del lector. Para B1, el coste por turno es el de reenviar todo el historial. Se calcula con la tabla de precios fechada y se muestra la fórmula junto a la cifra.

**Latencia** (medida en el cliente, percentiles sobre el histograma completo, nunca medias de percentiles)

- `recall` p50 / p95 / p99, con y sin reranker.
- Confirmación de escritura p50 / p95.
- Tiempo de extracción por sesión p50 / p95.
- Parámetros de carga declarados junto a la cifra: máquina, número de hechos y episodios del sujeto, concurrencia (1 y 8 clientes), embebedor local o remoto.

**Tamaño**

- Hechos activos, sustituidos y archivados por sujeto; bytes por sujeto.

**Seguridad**

- **Tasa de aceptación de inyecciones**: casos de la suite de envenenamiento en los que el recuerdo plantado aparece en un `recall` por defecto. Contramétrica SM-C6.
- **Filas ajenas** en la suite de aislamiento. Debe ser 0.

## 6. Tabla de resultados

La tabla que irá al README. Todas las celdas están pendientes.

| Brazo | Exactitud (500) | Exactitud (440 retenidas) | `knowledge-update` | `temporal-reasoning` | `multi-session` | Abstención | Tokens del lector / pregunta | Coste / turno (USD) | `recall` p50 / p95 (ms) |
|---|---|---|---|---|---|---|---|---|---|
| B0 sin memoria | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | n/a |
| B1 contexto completo | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | n/a |
| B2 top-k vectorial | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente |
| **M mnemo** | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente | pendiente |

Hipótesis registradas (del PRD): M ≥ B2 + 8 puntos; M ≥ B1; M ≥ B2 + 10 puntos en `knowledge-update` y `temporal-reasoning`; tokens del lector ≤ 8.000; `recall` p95 ≤ 400 ms sin reranker; abstención de M no inferior a la de B1.

Si una hipótesis no se cumple, la tabla se publica igual y la sección de análisis explica por qué.

## 7. Cifras publicadas por terceros (contexto, no comparación)

Ninguna de estas cifras se reprodujo. Usan lectores, jueces, presupuestos y protocolos distintos entre sí y distintos del de este plan.

| Sistema | Benchmark | Cifra | Lector | Fuente | Fiabilidad |
|---|---|---|---|---|---|
| Contexto completo | LongMemEval_S | 60,6 % (87,0 % en oráculo) | GPT-4o | [artículo de LongMemEval](https://arxiv.org/html/2410.10813v2) | Primaria, autores del benchmark |
| Contexto completo | LongMemEval | 60,2 % / 55,4 % | GPT-4o / GPT-4o-mini | [Zep](https://www.getzep.com/blog/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/) | Primaria, proveedor |
| Zep | LongMemEval | 71,2 % / 63,8 % | GPT-4o / GPT-4o-mini | igual | Primaria, proveedor, sobre su propio producto |
| Mem0 (plataforma gestionada) | LongMemEval | 94,4 | no indicado en el README | [README de Mem0](https://github.com/mem0ai/mem0) | Primaria, proveedor; el README advierte que no es la versión de código abierto |
| Mem0 (plataforma gestionada) | LoCoMo | 92,5 | no indicado en el README | igual | igual |
| Mem0 base / grafo | LoCoMo | 66,88 % / 68,44 % | — | Citadas por [Zep](https://www.getzep.com/blog/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/) del artículo de Mem0 | Secundaria |
| Zep | LoCoMo | 75,14 % ± 0,17 (corregida por Zep) | — | igual | Primaria, proveedor; disputada por Mem0 |
| Contexto completo | LoCoMo | ≈ 73 % | GPT-4o-mini | igual | Primaria, proveedor |
| Mastra (memoria observacional) | LongMemEval | 84,23 % frente a 60,20 % de contexto completo | GPT-4o | Citada por [Penfield Labs](https://penfieldlabs.substack.com/p/we-audited-locomo-64-of-the-answer) | Secundaria |
| Letta | — | Sin cifra en su README | — | [README de Letta](https://github.com/letta-ai/letta) | — |

Dos lecturas de esta tabla. Primera: la cifra más alta sobre LoCoMo (92,5) queda a un punto del techo de ≈ 93,6 % que la auditoría calcula para un sistema perfecto, con un juez que la misma auditoría mostró laxo; a esa altura la cifra dice más del benchmark y del juez que del sistema. Segunda: mnemo, con un lector barato, quedará previsiblemente por debajo de las cifras más altas. El argumento del proyecto no es ganarles, es que cualquiera puede comprobar su número.

## 8. Validación del juez

El juez es un modelo y se equivoca. Antes de publicar:

1. Se extraen 100 respuestas del brazo M (estratificadas por tipo, mitad dadas por correctas y mitad por incorrectas por el juez).
2. Las etiqueto a mano, sin ver el veredicto del juez, en un CSV.
3. Se reportan la tasa de verdaderos positivos y la de verdaderos negativos del juez frente a mis etiquetas, y los desacuerdos se listan.
4. Si alguna de las dos tasas baja del 90 %, el titular se publica con esa advertencia junto a la cifra y con la exactitud corregida por mis etiquetas en la muestra.

Con un solo anotador no hay acuerdo entre anotadores que reportar; se declara como limitación.

## 9. Análisis de fallos

Lo más valioso del proyecto después de la tabla. Método:

1. Leer al menos 100 trazas del brazo M: todas las falladas del subconjunto de desarrollo y una muestra de las retenidas tras la corrida final.
2. Anotación abierta: una frase por traza sobre qué salió mal.
3. Agrupar en una taxonomía. Categorías de partida, que la lectura puede cambiar: hecho no extraído · hecho extraído mal · sustitución que no ocurrió · sustitución indebida · entidad mal resuelta · evidencia no recuperada · evidencia recuperada pero cortada por el presupuesto · el lector tenía la evidencia y falló · el juez se equivocó · la clave del benchmark es dudosa.
4. **Matriz de primera etapa fallida**: filas = última etapa correcta (ingesta, extracción, resolución, recuperación, ensamblado, lectura), columnas = tipo de pregunta. Se corrige primero la etapa más temprana con más fallos.
5. Publicar en `docs/failure-analysis.md` la taxonomía con recuentos, tres ejemplos por categoría y qué se cambió a raíz de ello.

La comparación con el conjunto oráculo separa los dos grandes grupos: si M falla donde el oráculo acierta, el problema es de recuperación; si ambos fallan, es del lector o de la pregunta.

## 10. Presupuesto

Estimación propia, a precio de lista del modelo por defecto. Se sustituye por la medición de M1.

| Concepto | Cálculo | USD |
|---|---|---|
| Ingesta de una sesión (~2.900 tokens) | extracción ≈ 3.500 entrada + 500 salida; veredictos ≈ 1.500 entrada + 200 salida | ≈ 0,0023 |
| Ingesta completa (500 preguntas × ~40 sesiones) | 20.000 sesiones × 0,0023, sin caché | ≈ 46 |
| B1 contexto completo | 500 × 115.000 tokens de entrada | ≈ 14 |
| Lectura de B0, B2, M y ablaciones de lectura | ~12 brazos × 500 × ~6.000 tokens | ≈ 9 |
| Ingesta de desarrollo (60 preguntas), por cada reingesta | 2.400 sesiones × 0,0023 | ≈ 5,5 |
| Embeddings y reranker | local, o céntimos alojado | < 1 |
| Juez | no estimado: precio sin verificar | Q-2 |

La suma a precio de lista (≈ 70 más el juez y las reingestas de desarrollo) **supera el tope de USD 60**. Formas de cerrar la brecha, en orden de preferencia: (1) la caché de sesiones, si el supuesto A-6 se cumple; (2) el precio por lotes, a mitad, para la ingesta final; (3) la capa gratuita del proveedor repartida en varios días; (4) reducir la corrida final a un subconjunto estratificado de 250 preguntas y declararlo. La decisión se toma al cerrar M1, con el coste por sesión ya medido (Q-1).

## 11. Reproducción

Comando objetivo, sobre un commit etiquetado:

```
docker compose up -d
uv run --project eval mnemo-eval fetch
uv run --project eval mnemo-eval run --protocol eval/config/protocol.yaml --arms all
uv run --project eval mnemo-eval report --out eval/results/
```

Se commitea `eval/results/<protocolo>/summary.json` y la tabla en Markdown. Las trazas completas se publican como artefacto de la versión, no en el repositorio.

## 12. Amenazas a la validez

- **Lector barato.** Las cifras absolutas no son comparables con las de la sección 7. Lo que sí es válido es la comparación entre brazos.
- **LongMemEval_S cabe en contexto.** Por eso B1 está en la tabla y el coste por turno acompaña a la exactitud.
- **Ajuste sobre 60 preguntas.** Mitigado con las 440 retenidas y SM-C3.
- **Un solo anotador** en la validación del juez y en las suites propias.
- **Suites propias pequeñas** (30 y 20 casos): sirven de regresión, no de estimación.
- **Datos en inglés.** No se afirma nada sobre español.
- **Posible contaminación.** Los conjuntos son públicos desde 2024; un lector puede haberlos visto. B0 lo acota: si B0 acierta mucho, hay fuga.
- **Historiales sintéticos.** Ambos benchmarks usan conversaciones generadas o guiadas; la frecuencia real de cada tipo de fallo en producción no se deduce de aquí.

## Registro de cambios del protocolo

| Fecha | Cambio | Motivo | Brazos reejecutados |
|---|---|---|---|
| 2026-11-19 | Versión inicial | — | — |

# ADR-0004: LongMemEval_S como benchmark titular y protocolo registrado

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** de un solo sentido una vez publicado un número. Cambiar de benchmark o de protocolo después invalida la comparación con lo ya publicado y parece elegir el resultado.
- **Coste de revertir:** una nueva corrida completa (unos USD 25-50 y dos semanas) y la credibilidad del primer número.
- **Disparador de revisión:** el conjunto de datos deja de estar disponible; se publica un defecto que lo invalida; o el coste medido en M1 hace imposible la corrida de 500 preguntas dentro del presupuesto.

## Contexto
El proyecto vale por un número en un benchmark reconocido. Los candidatos al 2026-11-19:

| Benchmark | Tamaño | Qué mide | Licencia | Problemas conocidos |
|---|---|---|---|---|
| **LoCoMo** | 10 conversaciones publicadas (de 50 originales); 300 turnos y 9.000 tokens de media, hasta 35 sesiones ([resumen del artículo](https://arxiv.org/abs/2402.17753), [repositorio](https://github.com/snap-research/locomo)) | QA de un salto, varios saltos, temporal, dominio abierto y adversarial | CC BY-NC 4.0 ([LICENSE.txt](https://raw.githubusercontent.com/snap-research/locomo/main/LICENSE.txt)) | Una auditoría halló 99 errores en 1.540 preguntas (6,4 %), techo teórico ≈ 93,6 %, y un juez `gpt-4o-mini` que aceptó el 62,81 % de respuestas deliberadamente incorrectas pero del tema correcto ([Penfield Labs](https://penfieldlabs.substack.com/p/we-audited-locomo-64-of-the-answer)). Zep señala que no evalúa actualización de conocimiento, que la categoría 5 carece de respuestas de referencia y que una línea base de contexto completo alcanza ≈ 73 % ([Zep](https://www.getzep.com/blog/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/)) |
| **LongMemEval** | 500 preguntas; variante S con ~115.000 tokens (~40 sesiones) por pregunta, variante M con ~500 sesiones ([repositorio](https://github.com/xiaowu0162/LongMemEval)) | Extracción de información, razonamiento multi-sesión, actualización de conocimiento, razonamiento temporal y abstención | Repositorio MIT; la página no declara licencia aparte para los datos | La variante S cabe en las ventanas de contexto actuales, así que mide más gestión de contexto que memoria a muy largo plazo ([Penfield Labs](https://penfieldlabs.substack.com/p/we-audited-locomo-64-of-the-answer)). Historial limpiado en septiembre de 2025 |
| **LongMemEval-V2** | 451 preguntas sobre trayectorias de agentes web, hasta 115 M de tokens, multimodal ([página](https://xiaowu0162.github.io/longmemeval-v2/)) | Memoria de experiencia de agentes | No declarada en la página | Otra modalidad y otra escala; tabla pública "próximamente" |
| **LoCoMo-Plus**, **BEAM** | Vistos solo en resúmenes de búsqueda y material de proveedores | — | Sin verificar | Sin verificar; LoCoMo-Plus hereda las 1.540 preguntas de LoCoMo con sus errores ([Penfield Labs](https://penfieldlabs.substack.com/p/we-audited-locomo-64-of-the-answer)) |

La disputa pública entre proveedores añade contexto: el artículo de Mem0 atribuyó a Zep un 65,99 % en LoCoMo; Zep respondió con una cifra propia que luego corrigió a 75,14 % ± 0,17 reconociendo un error de cálculo ([Zep](https://www.getzep.com/blog/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/)). Según resúmenes de terceros, la cifra inicial de Zep rondaba el 84 % y Mem0 obtuvo 58,44 % al reejecutar el mismo código; ninguna de esas dos cifras se verificó en una fuente primaria. Cada parte midió con la configuración que le favorecía.

## Impulsores de la decisión
* **Imprescindible** — el benchmark mide lo que mnemo afirma hacer distinto: hechos que cambian y preguntas temporales.
* **Imprescindible** — un juez con acuerdo humano documentado.
* **Imprescindible** — reproducible por un tercero con un comando y poco dinero.
* **Deseable** — comparabilidad con cifras ya publicadas.
* **Deseable** — licencia que no limite el uso posterior.

## Decisión
1. **Titular: exactitud en LongMemEval_S** (versión limpia, las 500 preguntas), con el desglose por tipo. Es el único candidato que evalúa actualización de conocimiento, razonamiento temporal y abstención; su juez (`gpt-4o-2024-08-06` con prompts por tipo) tiene más del 97 % de acuerdo con expertos humanos según el artículo ([LongMemEval](https://arxiv.org/html/2410.10813v2)).
2. **Secundario: LoCoMo**, categorías 1 a 4 (1.540 preguntas), reportado junto a sus defectos conocidos y el techo de ≈ 93,6 %. Se incluye porque sigue siendo la cifra más citada, no porque sea la mejor medida.
3. **Protocolo registrado antes de medir** en [evaluation-plan.md](../evaluation-plan.md): mismos lector, juez, prompt de lectura y presupuesto de tokens para todos los brazos; tres líneas base propias (sin memoria, contexto completo, top-k vectorial); 60 preguntas estratificadas para ajustar y 440 retenidas; se reportan las 500 y las 440; modelos y precios fijados con fecha.
4. **Las cifras de terceros no entran en la tabla principal.** Se listan aparte, con fuente, lector y juez, y la advertencia de que no son comparables.
5. La objeción "cabe en la ventana de contexto" se responde midiendo: la línea base de contexto completo está en la tabla, y el argumento de mnemo es la exactitud por token, no solo la exactitud.

## Consecuencias
### Positivas
- El número titular está alineado con el diferenciador (ADR-0002).
- Las líneas base propias hacen el resultado interpretable aunque nadie crea la comparación con terceros.
- Protocolo escrito de antemano: no hay margen para elegir la métrica después de ver los datos.

### Negativas
- LongMemEval_S cuesta: 500 historiales de ~115.000 tokens son unos 57,5 M de tokens solo de ingesta. Es el riesgo R-2.
- El lector barato hace que las cifras absolutas no sean comparables con las publicadas con GPT-4o.
- LoCoMo como secundario obliga a explicar sus defectos cada vez que se cita.

### Riesgos
- El juez `gpt-4o-2024-08-06` deja de estar disponible → juez alternativo con los mismos prompts, validado contra 100 etiquetas propias (se hace en cualquier caso).
- El subconjunto de desarrollo filtra información al ajustar umbrales → contramétrica SM-C3 y reporte separado de las 440 retenidas.
- La licencia de los datos de LongMemEval resulta restrictiva → no se redistribuyen en ningún caso; pregunta abierta Q-5.

### Puntos de sensibilidad
- El modelo lector mueve todas las cifras absolutas; cambiarlo obliga a reejecutar todos los brazos.
- El presupuesto de tokens del contexto mueve exactitud y coste a la vez.

### Puntos de compromiso
- **Comparabilidad contra coste**: usar GPT-4o como lector acercaría las cifras a las publicadas, pero la línea base de contexto completo por sí sola costaría más que todo el presupuesto.
- **Credibilidad contra amplitud**: un benchmark bien hecho en vez de cuatro a medias.

## Alternativas consideradas
1. **LoCoMo como titular** — rechazada: es el más citado, pero con 6,4 % de la clave errónea, un juez laxo documentado, sin preguntas de actualización de conocimiento y licencia no comercial.
2. **LongMemEval_M como titular** — rechazada por coste: ~500 sesiones por pregunta multiplica la ingesta por más de diez. Queda como trabajo futuro.
3. **LongMemEval-V2** — rechazada: evalúa memoria de trayectorias de agentes web, multimodal; no es el problema de mnemo.
4. **Conjunto propio en español** — rechazada como titular: no es un benchmark reconocido. La suite propia se limita a envenenamiento (30 casos) y a casos de validez temporal, y se presenta como tal.

## Notas de implementación
- `eval/config/protocol.yaml` fija conjunto, versión y suma de verificación de los datos, lector, juez, prompts, presupuesto de tokens y semilla. Cambiarlo cambia el identificador del protocolo que aparece en la tabla.
- Función de aptitud: `mnemo-eval verify-protocol` falla si un archivo de resultados mezcla brazos de protocolos distintos.
- Los datos se descargan a `eval/data/`, ignorado por git.

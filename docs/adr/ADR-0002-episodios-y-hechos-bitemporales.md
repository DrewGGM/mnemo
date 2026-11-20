# ADR-0002: Episodios inmutables y hechos bitemporales que se sustituyen

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** de un solo sentido. El modelo de memoria define el esquema, la API pública (`as_of`, `timeline`, estados) y lo que el benchmark mide.
- **Coste de revertir:** unas 5 semanas y una nueva ingesta completa del benchmark.
- **Disparador de revisión:** en M1 los hechos extraídos no mejoran al top-k vectorial en al menos 3 puntos sobre las 60 preguntas de desarrollo; o el análisis de fallos atribuye más del 30 % de los errores a sustituciones incorrectas.

## Contexto
La pregunta de fondo es qué pasa cuando un hecho cambia ("me mudé de Bogotá a Medellín"). Los proyectos existentes responden distinto:

- Graphiti invalida el hecho anterior en vez de borrarlo y le mantiene una ventana de validez ([README](https://github.com/getzep/graphiti)).
- El README de Mem0 describe hoy una extracción de un solo paso que solo añade ("ADD-only"), sin UPDATE ni DELETE, y delega en la recuperación el ordenar por tiempo ([README](https://github.com/mem0ai/mem0)).

LongMemEval evalúa de forma explícita actualización de conocimiento y razonamiento temporal ([repositorio](https://github.com/xiaowu0162/LongMemEval)), y su artículo muestra dos cosas que condicionan el modelo: indexar con hechos como clave adicional mejora el recall@k un 9,4 % y la exactitud un 5,4 % en promedio, pero reemplazar las rondas por hechos descompuestos perjudica el rendimiento global ([artículo](https://arxiv.org/html/2410.10813v2)). Es decir: los hechos ayudan a encontrar, y el texto original sigue haciendo falta para leer.

## Impulsores de la decisión
* **Imprescindible** — responder "qué es cierto hoy" y "qué era cierto en tal fecha".
* **Imprescindible** — no perder nunca información por una decisión de un modelo; toda decisión debe poder auditarse y revertirse.
* **Imprescindible** — procedencia: cada hecho apunta a los episodios de los que salió.
* **Deseable** — ablación limpia entre memoria episódica y semántica.

## Decisión
Dos niveles de memoria, ambos por sujeto:

1. **Episodios** (memoria episódica): cada mensaje, inmutable, con fecha del hecho, sesión, rol y nivel de confianza. Se recuperan directamente y sirven de evidencia.
2. **Hechos** (memoria semántica): afirmaciones atómicas `sujeto – predicado – objeto` con enunciado en lenguaje natural, enlazadas a sus episodios de origen. Cuando el objeto es una entidad, el hecho es una arista del grafo.

Cada hecho lleva **dos líneas de tiempo**: la del mundo (`valid_from`, `valid_to`) y la del sistema (`recorded_at`, `invalidated_at`). Un conflicto no borra ni reescribe: el hecho anterior pasa a `superseded`, se cierra su `valid_to` y se enlaza con su sucesor mediante `superseded_by`; ambos comparten `chain_id`. Los estados y transiciones legales son los del diagrama de [architecture.md](../architecture.md#ciclo-de-vida-de-un-hecho). El modelo de lenguaje propone un veredicto; un núcleo puro (`internal/resolve`) decide qué transiciones se aplican, y un índice único parcial en la base impide dos hechos activos en la misma cadena.

## Consecuencias
### Positivas
- Las preguntas temporales se responden con un filtro, no con una inferencia.
- Una sustitución equivocada se corrige: los datos siguen ahí.
- Procedencia completa para citas, depuración y análisis de fallos.
- Ablaciones naturales: solo episodios, episodios y hechos, con y sin filtro de validez.

### Negativas
- La tabla de hechos solo crece (hasta que actúa el olvido o una purga).
- Dos llamadas al modelo por lote en escritura: coste y latencia de ingesta mayores que guardar y embeber.
- La API es más ancha: `as_of`, `include_history`, cinco estados.

### Riesgos
- El modelo marca como contradicción lo que es un matiz ("trabajo en Bogotá" y "vivo en Medellín") → el veredicto se limita a hechos con la misma entidad-sujeto y se revisa en el análisis de fallos; la retractación y la liberación manual existen para corregir.
- Fechas de validez mal resueltas → por defecto se usa la fecha del episodio, que es un dato y no una inferencia.

### Puntos de sensibilidad
- La calidad del prompt de extracción mueve directamente SM-1 y SM-3.
- El umbral de similitud para buscar vecinos en conflicto decide cuántas llamadas al modelo se hacen.

### Puntos de compromiso
- **Integridad temporal contra coste por turno**: más tokens en escritura a cambio de no perder historia.
- **Auditabilidad contra borrabilidad**: conservar todo choca con el derecho de supresión; se resuelve con la purga física por sujeto (FR-16), que sí borra las cadenas completas.

## Alternativas consideradas
1. **Solo añadir y ordenar por recencia en la lectura** — rechazada: más simple y barata, pero deja la decisión de qué está vigente al lector en cada consulta y no permite filtrar por fecha. Se conserva como brazo de ablación (A4 apagado).
2. **Actualizar o borrar en sitio (ADD/UPDATE/DELETE)** — rechazada: pierde la historia y hace irreversible un error del modelo.
3. **Solo episodios, sin extracción** — rechazada como diseño final, conservada como línea base B2 y como plan B de R-1.
4. **Event sourcing completo** — rechazada: la historia que importa ya queda en las propias filas; un registro de eventos aparte duplicaría el almacenamiento sin responder ninguna pregunta nueva.

## Notas de implementación
- `create unique index … on facts (tenant_id, chain_id) where status = 'active'`.
- `internal/resolve` no importa nada de E/S; se construye con pruebas primero y pruebas de propiedades: para cualquier secuencia de candidatos y veredictos, a lo sumo un activo por cadena, ningún hecho desaparece, y `valid_to ≥ valid_from`.
- La transición se aplica con `SELECT … FOR UPDATE` sobre el hecho activo de la cadena; el perdedor de una carrera reintenta sobre el estado nuevo.

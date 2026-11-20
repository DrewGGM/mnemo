# mnemo — preguntas abiertas y supuestos

Estado: lista viva · Última revisión: 2026-11-20

Aquí va lo que no está decidido, lo que una tarea debe anotar al terminar y lo aprendido que cambia el plan (reglas 3 y 7 de [plan/tasks.md](plan/tasks.md#cómo-se-ejecuta-este-archivo)). Una pregunta se cierra escribiendo su resolución con fecha en la columna **Estado**; no se borra.

## 1. Preguntas

| Q | Pregunta | ¿Bloquea? | Supuesto vigente | Responsable | Para | Estado |
|---|---|---|---|---|---|---|
| Q-1 | ¿Cuál es el tope real de gasto en APIs? La estimación a precio de lista (≈ USD 70 más el juez) supera el tope ([evaluation-plan.md §10](evaluation-plan.md#10-presupuesto)) | M1 | A-2: USD 60 | Autor | 2026-12-06; decisión final al cerrar T-011 | Abierta |
| Q-2 | ¿Qué modelo lector y qué juez se usan? El precio de `gpt-4o-2024-08-06` como juez no está verificado | M1 | A-3 | Autor | 2026-12-06 | Abierta |
| Q-3 | ¿Bajo qué ruta de módulo y cuenta se publica? | M0 | A-1: `github.com/DrewGGM/mnemo` | Autor | antes de T-001 | Abierta |
| Q-4 | Sin asignar: ningún documento la cita | — | — | — | — | — |
| Q-5 | ¿Qué licencia tienen los datos de LongMemEval? El repositorio es MIT y la página no declara otra | No | No se redistribuyen en ningún caso | Autor | antes de T-035 | Abierta |
| Q-6 | ¿Cómo afecta la licencia CC BY-NC 4.0 de LoCoMo a anunciar resultados en un nivel de pago? | No | No hay nivel de pago en esta fase | Autor | al definir un nivel alojado | Abierta |
| Q-7 | ¿Hay demanda concreta de un nivel alojado? Si la hubiera: obligaciones bajo la Ley 1581 de 2012 y la Circular Externa 002 de 2024 de la SIC; transferencia internacional a proveedores de modelos sin analizar | No | No se construye | Autor | si aparece demanda | Abierta |
| Q-8 | ¿Qué resumen (*digest*) fija la imagen `pgvector/pgvector:pg17` una vez verificada? | No | A-9 | Autor | T-002 | Abierta |
| Q-9 | ¿Qué herramienta de mutación para Go se usa en `internal/resolve` y `internal/retrieve/fuse.go`? Si ninguna es razonable, diez mutantes escritos a mano | No | — | Autor | elegir en T-018; aplicar antes de T-036 | Abierta |
| Q-10 | ¿OAuth, JWT o rotación automática de claves? (fuera de alcance de T-012; ADR-0003, alternativa 4; riesgo aceptado AR-2) | No | Credencial `Bearer` estática | Autor | cuando un cliente lo exija | Abierta |
| Q-11 | ¿Correferencia con modelo o fusión manual de entidades? (fuera de alcance de T-017) | No | Trigramas ≥ 0,6 dentro del sujeto | Autor | si el análisis de fallos atribuye > 15 % de los errores a entidades ([architecture.md §10](architecture.md#10-lo-que-deliberadamente-no-se-construye-todavía)) | Abierta |
| Q-12 | ¿Consultas por tiempo de sistema (`recorded_at`, `invalidated_at`)? (fuera de alcance de T-020) | No | Solo `as_of` sobre la validez del mundo | Autor | sin disparador definido | Abierta |
| Q-13 | ¿Cómo hace el rol `mnemo_app`, sin `BYPASSRLS` y con RLS forzada, las operaciones que no conocen el inquilino de antemano: buscar una credencial por su hash (`store.LookupKey`, T-012), tomar trabajos "por turnos entre inquilinos" (`store.ClaimJob`, T-015) y la limpieza diaria (T-016)? Sin `app.tenant_id` la consulta falla cerrada (ADR-0003) | T-012 | A-13 | Autor | antes de T-012 | Abierta |
| Q-14 | T-012 y T-029 consumen `store.LookupKey` desde `internal/httpapi` e `internal/mcpserver`, y el `archcheck` de T-003 les prohíbe importar `internal/store`. Además T-003 permite `internal/config` y [architecture.md §5](architecture.md#5-módulos) dice "solo importan el paquete raíz". ¿Cuál es la regla? | T-003, T-012 | A-14 | Autor | antes de T-003 | Abierta |
| Q-15 | `fact_sources` tiene clave primaria `(fact_id, episode_id)` y `episode_id … on delete set null`: una columna de clave primaria no admite nulos, así que borrar un episodio con hechos falla. FR-14 (T-026) borra episodios tras resumirlos. ¿Qué pasa con la procedencia? | T-010 | Ninguno | Autor | antes de T-010 | Abierta |
| Q-16 | `facts.subject_entity_id` y `object_entity_id` referencian `entities`, pero `0002_facts.sql` (T-010) es anterior a `0007_entities.sql` (T-017) | No | A-15 | Autor | T-010 | Abierta |
| Q-17 | Varias listas "Tocar solo" no incluyen archivos que su "Hecho cuando" o sus "Interfaces" necesitan (detalle en [plan/README.md](plan/README.md#34-hallazgos), F-5). La regla 5 del plan impide ampliarlas en la propia tarea. ¿Se corrigen las listas antes de empezar? | T-002 | Ninguno | Autor | antes de T-002 | Abierta |
| Q-18 | T-025 exige que `release` deje fila en `audit_log`, que crea T-028; T-025 no depende de T-028. ¿Se reordena o se mueve la tabla? | T-025 | Ninguno | Autor | antes de T-025 | Abierta |
| Q-19 | Ninguna tarea añade al CI las etapas de Python (`ruff`, `pytest`), `pip-audit`, `uv sync --frozen`, la prueba de punta a punta ni la evaluación mínima de [quality-baseline.md §7](quality-baseline.md#7-integración-continua); ninguna crea el gancho de pre-commit de §2; ninguna declara `pip-audit` en `eval/pyproject.toml`. Afecta al punto abierto de la compuerta de seguridad | No | — | Autor | antes de cerrar M0 | Abierta |
| Q-20 | Ninguna tarea nombra una prueba `TestFR9_…`, y `go run ./tools/fr-trace -from 1 -to 23` (T-031) falla sin ella | T-031 | — | Autor | antes de T-021 | Abierta |
| Q-21 | Dos funciones de aptitud de ADR sin tarea: el arranque falla si el modelo de embeddings configurado no coincide con el registrado (ADR-0006, decisión 4); el script de dependencias falla si `go.mod` gana un cliente de otra base de datos (ADR-0001) | No | — | Autor | antes de cerrar M3 | Abierta |
| Q-22 | `idempotency_keys` no tiene `subject_id`, pero guarda respuestas con datos del sujeto y la supresión las incluye ([security-baseline.md §6](security-baseline.md#6-datos-personales-y-supresión), TH-13). `TestPurge_NoRowsLeftAnyTable` descubre tablas por la columna `subject_id` y no la vería. Al revés, `api_keys.subject_id` haría que la purga borre las credenciales ligadas al sujeto sin que ningún documento lo diga | T-028 | Ninguno | Autor | antes de T-028 | Abierta |
| Q-23 | ¿En qué idioma van los textos que ve un cliente: etiquetas del contexto (`[hecho · válido …]` en [architecture.md §7](architecture.md#7-api-del-servicio)), mensajes de error del sobre y de las herramientas MCP (ejemplo en español en T-029)? | No | A-12 | Autor | antes de T-013 | Abierta |
| Q-24 | El plan fija "un commit por tarea" y la salida de Verificar en el commit o en el PR, pero no el formato del mensaje ni el nombre de las ramas | No | A-11 | Autor | antes de T-001 | Abierta |

## 2. Supuestos

Los supuestos A-1 a A-10 están en el [índice de supuestos del PRD](prd.md#assumptions-index); aquí solo se sigue quién los falsa. A-11 en adelante nacen de la comprobación de consistencia del 2026-11-20.

| Supuesto | Lo falsa o confirma | Estado |
|---|---|---|
| A-1 Ruta de módulo `github.com/DrewGGM/mnemo` | El autor, antes de T-001 | Sin confirmar |
| A-2 USD 60 de presupuesto | T-011 (coste medido) | Sin confirmar |
| A-3 Gemini 3.1 Flash-Lite como lector y extractor; `gpt-4o-2024-08-06` como juez | T-007, T-009 | Sin confirmar |
| A-4 Qwen3-Embedding-0.6B servible en local con API compatible con OpenAI | T-007 (prueba `live`) | Sin confirmar |
| A-5 Gemini con endpoint compatible y salida JSON con esquema | T-007 (prueba `live`) | Sin confirmar |
| A-6 Sesiones repetidas entre historiales de LongMemEval_S | T-008 (tasa de repetición) | Sin confirmar |
| A-7 Búsqueda exacta suficiente hasta 50.000 vectores por sujeto | T-032 (`TestRecall_P95`, variante de 50.000) | Sin confirmar |
| A-8 Sin segundo revisor | Aparición de un colaborador | Vigente |
| A-9 `pgvector/pgvector:pg17` con pgvector ≥ 0.8 | T-002 | Sin confirmar |
| A-10 8-10 h semanales sostenidas | Dos semanas seguidas por debajo de 5 h | Vigente |

- `[SUPUESTO A-11]` Una rama por tarea, `t-NNN-<slug>`, con un único commit, integrada en `main` por avance rápido con el CI en verde. Primera línea del commit: `T-NNN: <título de la tarea en el índice>`; cuerpo: el comando Verificar y su salida. Mensajes en español, como el plan. *Falsación:* el autor fija otra convención (Q-24).
- `[SUPUESTO A-12]` Identificadores, comentarios de documentación, mensajes de registro y textos que ve un cliente (etiquetas del contexto, errores del sobre y de MCP) en inglés: el benchmark, el README y los nombres del glosario en código son ingleses; los ejemplos en español de `docs/` son ilustraciones. *Falsación:* el autor decide otra cosa en Q-23, o la configuración por inquilino de texto completo (`spanish`) exige etiquetas traducidas.
- `[SUPUESTO A-13]` Las operaciones sin inquilino conocido se hacen con funciones `SECURITY DEFINER` propiedad de `mnemo_admin`, de superficie mínima (buscar una credencial por hash y devolver su inquilino; reclamar el siguiente trabajo y devolver su inquilino; limpieza por antigüedad), y todo lo demás sigue bajo `SET LOCAL app.tenant_id`. El rol `mnemo_app` sigue sin `BYPASSRLS`. Exige un ADR que complete ADR-0003. *Falsación:* el autor elige otra vía en Q-13 (por ejemplo, un pool aparte con el rol de migraciones para el trabajador).
- `[SUPUESTO A-14]` `internal/httpapi` e `internal/mcpserver` importan solo el paquete raíz e `internal/config`; la verificación de credenciales se expone desde el paquete raíz y su firma la fija T-012. *Falsación:* el autor decide en Q-14 permitir `internal/store` en esos paquetes y actualiza [architecture.md §5](architecture.md#5-módulos) y T-003.
- `[SUPUESTO A-15]` `0002_facts.sql` crea `subject_entity_id` y `object_entity_id` sin clave foránea, y `0007_entities.sql` añade las restricciones con `ALTER TABLE` (cambio aditivo, coherente con [architecture.md, migraciones](architecture.md#migraciones)). *Falsación:* el autor adelanta la tabla `entities` a T-010.

## 3. Anotaciones que deja una tarea

Huecos que el plan manda rellenar. Cada fila se completa con fecha y dato al cerrar la tarea.

| Tarea | Qué se anota | Cierra | Dato |
|---|---|---|---|
| T-002 | Resumen de la imagen fijada | Q-8, A-9 | pendiente |
| T-003 | Primera ejecución del CI en GitHub, en verde | Compuerta de seguridad (cadena de suministro) | pendiente |
| T-005 | Versión del SDK de MCP fijada y revisiones de protocolo que declara | — | pendiente |
| T-007 | Resultado de la prueba `live`: salida JSON con esquema y vector de 1.024 dimensiones | A-4, A-5 | pendiente |
| T-008 | Fracción de sesiones repetidas entre preguntas | A-6 | pendiente |
| T-011 | Decisión de M1: seguir; seguir con corrida final reducida; o plan B de R-1 | Q-1, A-2 | pendiente |
| T-023 | Formato real de respuesta del reranker | — | pendiente |
| T-029 | Revisión de protocolo negociada con el cliente del SDK | — | pendiente |
| T-032 | p95 de `recall` con 50.000 episodios | A-7 | pendiente |
| T-033 | Si la lista de errores de la auditoría de LoCoMo no es utilizable | — | pendiente |
| T-035 | Lo aprendido para la versión siguiente | — | pendiente |

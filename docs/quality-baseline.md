# mnemo — línea base de calidad y fiabilidad

Estado: definida, sin código · Última revisión: 2026-11-21 ([ADR-0007](adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md))

Cada regla de este documento es un comando que falla con código distinto de cero. Lo que no se puede ejecutar no es una regla. Todos los comandos asumen una shell POSIX (en Windows, Git Bash) desde la raíz del repositorio.

## 1. Funciones de aptitud

Una por característica de arquitectura, como mínimo.

| Característica | Función de aptitud | Comando | Cuándo | Umbral |
|---|---|---|---|---|
| **Evaluabilidad** (rectora) | Cada fuente de `recall` se apaga por petición y el resultado es determinista con el embebedor falso | `go test ./... -tags=integration -run 'TestFR9_SourcesToggle|TestRecall_Deterministic'` | cada cambio | 0 fallos |
| Evaluabilidad | Toda llamada a un modelo deja una fila en `usage_events` | `go test ./... -tags=integration -run TestUsage_EveryProviderCallRecorded` | cada cambio | 0 llamadas sin registrar |
| Evaluabilidad | Un archivo de resultados no mezcla protocolos | `uv run --project eval mnemo-eval verify-protocol eval/results` | cada cambio en `eval/` | 0 mezclas |
| Aislamiento | Ninguna tabla con `tenant_id` devuelve filas ajenas, ni con SQL sin filtro; solo las funciones `SECURITY DEFINER` aprobadas en ADR-0007 saltan RLS, `mnemo_app` no lee vistas que se evalúen con los permisos de `mnemo_admin`, y no tiene `TRUNCATE` (que no pasa por RLS) ni `TRIGGER` sobre ninguna tabla | `go test ./internal/store -tags=integration -run 'TestIsolation_|TestRLS_|TestAppRole_'` | cada cambio | 0 filas; 0 funciones no aprobadas; 0 vistas de dueño accesibles; 0 tablas con `TRUNCATE` o `TRIGGER` para `mnemo_app` |
| Integridad temporal | Propiedades de la máquina de estados de hechos | `go test ./internal/resolve -run 'TestProp_' -count=1` | cada cambio | 0 contraejemplos |
| Integridad temporal | Sustitución bajo concurrencia | `go test ./internal/store -tags=integration -run TestSupersede_Concurrent -count=20` | cada cambio | 0 cadenas con dos activos |
| Instalabilidad | Arranque con un comando | `docker compose up -d --wait && curl -fsS http://127.0.0.1:8080/readyz` | cada cambio en `main` | 200 en < 60 s |
| Instalabilidad | Presupuesto de dependencias y ningún cliente de otra base, cola o caché (lista negra desde T-031, ADR-0001) | `go run ./tools/depbudget -max 6` | cada cambio | ≤ 6 directas; 0 de la lista negra |
| Evaluabilidad | `store.EnsureSetting`, sobre la que `mnemo serve` y `mnemo mcp` comprueban al arrancar el modelo de embeddings registrado, falla nombrando ambos valores si el configurado es otro (ADR-0006, decisión 4; ADR-0007, decisión 11; T-016) | `go test ./internal/store -tags=integration -run TestSettings_EmbedModelMismatchFails` | cada cambio | 0 discrepancias aceptadas |
| Rendimiento | Latencia de `recall` sobre un sujeto con 5.000 hechos y 10.000 episodios, 8 clientes, embebedor falso | `go test ./test/perf -tags=perf -run TestRecall_P95 -timeout 10m` | antes de cada versión | p95 ≤ 400 ms |
| Borrabilidad | Tras la purga, ninguna tabla con `subject_id` conserva filas (se descubren por `information_schema`) | `go test ./internal/store -tags=integration -run TestPurge_NoRowsLeftAnyTable` | cada cambio | 0 filas |
| Modularidad | Los núcleos puros no importan E/S; HTTP y MCP solo importan el paquete raíz e `internal/config` | `go run ./tools/archcheck` | cada cambio | 0 violaciones |
| Seguridad | Sin vulnerabilidades alcanzables | `go tool govulncheck ./... && uv run --project eval pip-audit` | cada cambio | 0 |
| Contrato | Las rutas implementadas coinciden con `api/openapi.yaml` | `go test ./internal/httpapi -run TestContract_RoutesMatchOpenAPI` | cada cambio | 0 diferencias |
| Privacidad | Un texto centinela no aparece en registros ni métricas | `go test ./test/e2e -tags=e2e -run TestLogs_NoContentLeak` | cada cambio | 0 apariciones |
| Lectura sin modelo generativo | `recall` no llama a `ChatModel` | `go test ./... -tags=integration -run TestRecall_NeverCallsChatModel` | cada cambio | 0 llamadas |

`tools/archcheck` y `tools/depbudget` son programas Go de unas decenas de líneas con la biblioteca estándar (`go list -deps -json`). No son dependencias.

## 2. Sensores

| Regla | Comando | Lo aplica |
|---|---|---|
| Formato | `test -z "$(gofmt -l .)"` | gancho de pre-commit y CI |
| Análisis estático | `go vet ./... && go tool staticcheck ./...` | gancho de pre-commit y CI |
| Pruebas unitarias | `go test ./...` | gancho de pre-commit y CI |
| Pruebas de integración | `go test -tags=integration ./...` | CI |
| Cobertura | `go test -tags=integration -coverprofile=coverage.out ./... && go run ./tools/covergate -profile coverage.out` | CI (umbral por paquete y caída máxima de 2 puntos) |
| Arquitectura y dependencias | `go run ./tools/archcheck && go run ./tools/depbudget -max 6` | CI |
| Python | `uv run --project eval ruff check eval && uv run --project eval ruff format --check eval && uv run --project eval pytest eval/tests` | gancho de pre-commit y CI |
| Vulnerabilidades | `go tool govulncheck ./...` | CI |

`staticcheck` y `govulncheck` se fijan como herramientas en `go.mod` (directiva `tool`, disponible desde Go 1.24), de modo que su versión viaja con el repositorio y no cuentan como dependencias del presupuesto.

Se montan en T-001 y T-003, antes de cualquier funcionalidad. El gancho de pre-commit es `.githooks/pre-commit`, versionado: lo crea T-003 con las líneas de Go y T-006 le añade las de Python; se activa con `git config core.hooksPath .githooks` (ADR-0007, decisión 9).

## 3. Reglas de código

Solo lo que se aparta del estándar de Go o es específico de mnemo.

- **Nombres de dominio.** Prohibidos los paquetes `utils`, `helpers`, `common`, `shared`. Los términos del [glosario](prd.md#glossary) se usan tal cual en el código: `Episode`, `Fact`, `Entity`, `Scope`, `Supersede`, `Retract`, `Purge`. `Forget` es archivar por política; nunca borrar.
- **Núcleo puro, borde impuro.** `internal/resolve`, `internal/retrieve` e `internal/policy` no hacen E/S, no leen el reloj (reciben `now`) ni generan identificadores (los reciben).
- **SQL solo en `internal/store`**, siempre parametrizado, siempre dentro de una transacción con `app.tenant_id`.
- **Errores.** Nunca se ignoran. Los de dominio son valores exportados (`ErrNotFound`, `ErrConflict`, `ErrInvalidState`, `ErrForbidden`); los de infraestructura se envuelven con `%w` y contexto de operación e identificadores, nunca con contenido. `internal/httpapi` traduce a códigos del sobre en un único lugar.
- **Contexto.** Toda función que hace E/S recibe `context.Context` y lo respeta; toda llamada externa lleva tiempo límite.
- **Tamaño.** Funciones de hasta 50 líneas, archivos de hasta 400, complejidad ciclomática de hasta 10. Se revisa a mano; `staticcheck` cubre lo demás.
- **Comentarios.** Explican por qué. Las funciones exportadas del paquete raíz llevan comentario de documentación y, las principales, un `Example` ejecutable.
- **Prompts.** Archivos de texto en `internal/extract/prompts/`, con un identificador de versión en la primera línea. Cambiar un prompt cambia ese identificador, invalida la caché de extracción y obliga a reejecutar el subconjunto de desarrollo.
- **Sin estado global.** Ninguna variable de paquete mutable; la configuración se valida entera al arrancar y el proceso termina si es inválida.
- **Regla de tres** antes de extraer una abstracción. Ninguna interfaz con una sola implementación, salvo los tres puertos de modelos, que tienen la real y la falsa.

Lista de autorrevisión antes de cada commit (no hay segundo revisor, supuesto A-8): ¿cambió algo fuera de la lista "Tocar solo" de la tarea? · ¿hay SQL fuera de `internal/store`? · ¿algún registro incluye contenido? · ¿la prueba falló antes de escribir el código? · ¿el nombre nuevo está en el glosario?

## 4. Estabilidad

Todo punto de integración va a fallar. Qué pasa en cada uno:

| Punto de integración | Tiempo límite | Reintentos | Si falla |
|---|---|---|---|
| Embeddings en escritura | 5 s | 3, retroceso exponencial con fluctuación, solo 429 / 5xx / red | El episodio se guarda sin vector; un trabajo `embed` lo completa |
| Embeddings en lectura | 5 s | 1 | `recall` responde con léxico y grafo; `degraded: ["vector"]` |
| Modelo de chat (extracción, veredictos) | 30 s | 3 dentro de la llamada; el trabajo, hasta 5 intentos | Trabajo `failed`, visible; los episodios siguen recuperables |
| Reranker | 1,5 s | 0 | Orden fusionado; `degraded: ["rerank"]`. Cortacircuitos: 5 fallos seguidos lo abren 30 s |
| PostgreSQL | `statement_timeout` 5 s; adquisición del pool 2 s | 0 | 503 `UNAVAILABLE`; `/readyz` pasa a 503 |

Antipatrones revisados contra el diseño:

| Antipatrón | Dónde podría aparecer | Contramedida |
|---|---|---|
| Hilos bloqueados | Trabajador esperando al modelo | Concurrencia acotada (4) y tiempo límite por llamada |
| Respuestas lentas | Reranker lento en `recall` | Tope de 1,5 s y degradación |
| Conjuntos sin límite | Listado de hechos, exportación | Paginación por cursor con máximo de 200; la exportación transmite en flujo |
| Capacidades desequilibradas | La API acepta escrituras más rápido de lo que el trabajador extrae | La cola absorbe; cuota diaria por inquilino; métrica de antigüedad de la cola |
| Multiplicador de fuerza | Un bucle de reintentos del cliente multiplica el gasto en modelos | Idempotencia por clave natural: reenviar los mismos episodios no vuelve a extraer |
| Usuario ballena | Un inquilino que monopoliza al trabajador | Toma de trabajos por turnos entre inquilinos |
| Estado estable | Tablas que solo crecen | Limpieza diaria de `jobs`, `idempotency_keys`, `usage_events` |

**Zona de deformación:** si hay que sacrificar algo para mantener viva la lectura, se sacrifica en este orden: reranker → fuente de grafo → fuente vectorial. La lectura léxica sobre episodios es lo último que cae.

## 5. Objetivos de nivel de servicio

mnemo es autoalojado: no hay un servicio propio con disponibilidad que prometer. Los objetivos son los de la instalación de referencia y sirven de criterio de regresión.

| Indicador | Objetivo | Cómo se mide |
|---|---|---|
| Latencia de `recall` sin reranker | p95 ≤ 400 ms, p99 ≤ 800 ms | En el cliente; sujeto con 5.000 hechos y 10.000 episodios; 8 clientes |
| Latencia de confirmación de escritura | p95 ≤ 200 ms con embebedor local | En el cliente; lote de 2 mensajes |
| Trabajos de extracción completados | ≥ 99 % en ≤ 5 intentos | `jobs` por estado |
| Antigüedad del trabajo más viejo en cola | ≤ 60 s en carga normal | `mnemo_jobs_oldest_age_seconds` |
| Peticiones con error 5xx | ≤ 0,5 % | Contador por ruta y código |

**Política ante incumplimiento** (versión para una persona): si una versión etiquetada empeora el p95 de `recall` más de un 20 % respecto a la anterior, o si la suite de aislamiento o la de purga falla, no se fusiona nada más que la corrección hasta que vuelva a verde. Sin excepciones para el aislamiento y la purga. Para la latencia, la excepción se escribe en el registro de deuda con fecha de vencimiento.

## 6. Observabilidad

Cuatro señales doradas, por ruta y por tipo de trabajo, en `/metrics`:

- **Latencia:** `mnemo_http_request_duration_seconds{route,code}` y `mnemo_recall_stage_duration_seconds{stage}` (embed, candidates, fuse, rerank, assemble).
- **Tráfico:** `mnemo_http_requests_total{route,code}`.
- **Errores:** el mismo contador por código, más `mnemo_provider_errors_total{provider,kind}` y `mnemo_auth_failures_total{reason}`.
- **Saturación:** `mnemo_jobs_queued`, `mnemo_jobs_oldest_age_seconds`, conexiones del pool en uso.

Además: `mnemo_provider_tokens_total{provider,model,direction}` y `mnemo_recall_degraded_total{source}`.

Registro estructurado en JSON con `log/slog`: `request_id`, `tenant_id`, `api_key_prefix`, `route`, `code`, `duration_ms`, `job_id`. El `subject_id` no se registra en claro, sino los 8 primeros caracteres de su hash. **Nunca contenido.**

Los tiempos por etapa van también en la respuesta de `recall`, que es de donde los toma el harness.

Trazas distribuidas: diferidas (un solo proceso). Copias de seguridad y recuperación ante desastres: responsabilidad del operador; el README indicará que `pg_dump` cubre todo el estado.

## 7. Integración continua

Etapas, en GitHub Actions, con acciones fijadas por SHA:

1. **Estático:** formato, `go vet`, `staticcheck`, `ruff`.
2. **Unitarias:** `go test ./...` y `pytest`.
3. **Integración:** servicio PostgreSQL con pgvector; `go test -tags=integration ./...` con cobertura.
4. **Arquitectura y cadena de suministro:** `archcheck`, `depbudget`, `govulncheck`, `pip-audit`, `go mod verify`.
5. **Humo:** `docker compose up -d --wait`, `/readyz`, y la prueba de punta a punta.
6. **Evaluación mínima:** el harness contra dos conversaciones de juguete con modelos falsos, para comprobar que la tubería completa produce una tabla. Sin llamadas de pago en CI.
7. **Versión (solo etiquetas):** imagen de contenedor, SBOM, resumen de resultados adjunto.

T-003 monta las etapas 1 a 5 de Go; T-006 añade las de Python (`uv sync --frozen`, `ruff`, `pytest`, `pip-audit`), la prueba de punta a punta de la etapa 5 y la etapa 6 (ADR-0007, decisión 9); T-031 añade la cobertura y la trazabilidad; T-036, la etapa 7.

Las corridas reales del benchmark no van en CI: cuestan dinero y se lanzan a mano sobre un commit etiquetado.

Estrategia de despliegue: no hay entorno que desplegar. La "versión" es una etiqueta de git, una imagen y un binario.

## 8. Documentación obligatoria

| Documento | Se actualiza cuando |
|---|---|
| `README.md` | Cambia el arranque, la API o la tabla de resultados |
| `api/openapi.yaml` | Cambia cualquier ruta, antes que el código |
| `docs/adr/` | Se toma una decisión de un solo sentido |
| `docs/evaluation-plan.md` | Cambia el protocolo (con su registro de cambios) |
| `docs/failure-analysis.md` | Tras cada corrida completa |
| `CONTRIBUTING.md` | Cambia un comando o una convención |

## 9. Riesgos y deuda técnica

| ID | Deuda | Cuadrante | Interés | Plan de pago | Revisión |
|---|---|---|---|---|---|
| D-1 | Texto completo de PostgreSQL en lugar de BM25 | Prudente, deliberada | Posible pérdida de calidad léxica, sin medir | Medir en la ablación A2; decidir con ese dato | Cierre de M4 |
| D-2 | Sin índice ANN | Prudente, deliberada | Ninguno hasta 50.000 vectores por sujeto | Añadir HNSW si A-7 se falsa | Cierre de M4 |
| D-3 | Estimación de tokens por caracteres / 4 | Prudente, deliberada | Presupuesto de contexto impreciso (±20 %) | Sustituir por tokenizador si SM-C2 se incumple por esta causa | Cierre de M4 |
| D-4 | Sin trazas distribuidas | Prudente, deliberada | Depuración más lenta de problemas entre etapas | Añadir cuando haya un segundo proceso | Versión 1.0 |
| D-5 | Límite de tasa en memoria | Prudente, deliberada | No vale con varias réplicas | Mover a PostgreSQL | Al añadir réplicas |
| D-6 | Resolución de entidades por trigramas | Prudente, deliberada | Entidades duplicadas o fusionadas por error | Revisar con la taxonomía de fallos | Cierre de M4 |
| D-7 | Un solo anotador para validar el juez | Prudente, deliberada | Menor credibilidad de la validación | Pedir una segunda anotación de 50 casos a un tercero | Tras publicar |

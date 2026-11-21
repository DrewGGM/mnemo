---
artifact: implementation-plan
blueprint: docs/blueprint.md
status: ready
---
# Plan de implementación — mnemo

## Cómo se ejecuta este archivo

1. **Una tarea por sesión de trabajo, un commit por tarea.** Antes de empezar se lee `CONTRIBUTING.md`, el bloque de la tarea y solo los archivos de "Leer antes".
2. **Planificar primero** si la tarea toca más de dos archivos o el enfoque no está claro. Si el cambio cabe en una frase, se hace sin más.
3. **Implementar exactamente la lista "Hecho cuando".** Ni más ni menos. Una idea mejor va a `docs/open-questions.md`, no al código.
4. **Ejecutar "Verificar", leer la salida completa y el código de salida, y pegar la salida en el mensaje del commit o en la descripción del PR.** Sin esa evidencia la tarea no está hecha. "Debería funcionar" no es un estado.
5. **Nunca ampliar "Tocar solo".** Si hace falta otro archivo, se abre una tarea nueva.
6. **Dos correcciones fallidas sobre el mismo problema:** se deja de parchear, se descarta el cambio y se vuelve a empezar la tarea con el problema mejor planteado.
7. **Lo aprendido que cambia el plan** se anota en `docs/open-questions.md` y la tarea se replantea aquí. No se ensancha en silencio.
8. Hecho todo, se marca la casilla del índice.

Los comandos asumen una shell POSIX (Git Bash en Windows), Go, Docker y `uv` instalados, y se ejecutan desde la raíz del repositorio. Las pruebas con etiqueta `integration` necesitan `docker compose up -d --wait db`.

En los componentes de criticidad ALTA (`internal/resolve`, aislamiento, autenticación, purga) la prueba se escribe y se ve fallar antes que el código.

Las listas "Tocar solo", dependencias y criterios de este archivo ya incorporan las decisiones de [ADR-0007](../adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md) (2026-11-21). Un criterio que exige modelos reales, gasto o etiquetado humano (T-007, T-009, T-011, T-023, T-034, T-035) se deja pendiente del autor en [open-questions.md §3](../open-questions.md#3-anotaciones-que-deja-una-tarea) con el comando que lo completa, y la tarea se marca con la casilla hecha y el sufijo "· parcial (open-questions §3)" (ADR-0007, decisión 14).

## Índice

### Previa
- [x] T-000 Resolver las decisiones abiertas que bloquean el plan ([ADR-0007](../adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md))

### M0 — Esqueleto andante
- [ ] T-001 Módulo Go, binario `mnemo serve` y `/healthz`
- [ ] T-002 PostgreSQL en Docker, migraciones y `/readyz`
- [ ] T-003 Imagen, CI y comprobadores de arquitectura y dependencias
- [ ] T-004 Guardar y recuperar episodios de punta a punta (solo vector)
- [ ] T-005 Servidor MCP por stdio con `recall`
- [ ] T-006 Harness de evaluación mínimo con modelos falsos

*Punto de control: una petición real atraviesa API → dominio → PostgreSQL → respuesta, por REST y por MCP; el CI la ejecuta; el harness produce una tabla.*

### M1 — Mayor riesgo: ¿la extracción aporta?
- [ ] T-007 Adaptadores de chat y embeddings compatibles con OpenAI
- [ ] T-008 Descarga y carga de LongMemEval_S y subconjunto de desarrollo
- [ ] T-009 Líneas base B0, B1, B2, lector, juez e informe
- [ ] T-010 Extracción de hechos (solo añadir) y recuperación de hechos
- [ ] T-011 Experimento M1 sobre 60 preguntas y decisión

*Punto de control: hay una tabla real con B0, B1, B2 y mnemo sobre 60 preguntas, con coste medido por sesión. Se decide seguir, recortar o pivotar.*

### M2 — Cimientos (ninguna historia de M3 empieza antes de cerrar M2)
- [ ] T-012 Inquilinos, seguridad a nivel de fila, credenciales y autenticación
- [ ] T-013 Contrato OpenAPI, sobre de error, validación y límites
- [ ] T-014 Idempotencia de escritura
- [ ] T-015 Cola de trabajos y extracción asíncrona
- [ ] T-016 Registro de consumo, tiempos por etapa y `/metrics`

### M3 — Historias

*Orden de ejecución: T-028 antes que T-025, que depende de su tabla `audit_log` (ADR-0007, decisión 7).*

- [ ] T-017 Entidades y aristas del grafo
- [ ] T-018 Núcleo puro de resolución de conflictos
- [ ] T-019 Resolución integrada: veredicto del modelo y transacción
- [ ] T-020 Consulta con fecha de referencia y línea de tiempo
- [ ] T-021 Fuente léxica y fusión por rango recíproco
- [ ] T-022 Fuente de grafo
- [ ] T-023 Reranker con degradación
- [ ] T-024 Ensamblado del contexto y ausencia de evidencia
- [ ] T-025 Niveles de confianza, cuarentena y suite de envenenamiento
- [ ] T-026 Consolidación
- [ ] T-027 Olvido por decaimiento
- [ ] T-028 Purga, retractación y exportación
- [ ] T-029 Servidor MCP completo y transporte HTTP
- [ ] T-030 API pública del paquete Go con ejemplos ejecutables
- [ ] T-031 Compuerta de cobertura y trazabilidad de requisitos

### M4 — Evaluación y publicación
- [ ] T-032 Latencia, coste por turno y ablaciones en el harness
- [ ] T-033 LoCoMo como benchmark secundario
- [ ] T-034 Validación del juez con 100 etiquetas
- [ ] T-035 Corrida completa, tabla de resultados y análisis de fallos
- [ ] T-036 Versión 0.1.0

---

## Previa

### T-000 · Resolver las decisiones abiertas que bloquean el plan   size:S  risk:LOW
- **Implementa:** ningún FR; desbloquea el plan · **Decisión:** ADR-0007
- **Depende de:** nada
- **Leer antes:** docs/plan/README.md#34-hallazgos · docs/open-questions.md
- **Tocar solo:** docs/adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md, docs/adr/README.md, docs/architecture.md, docs/blueprint.md, docs/open-questions.md, docs/plan/README.md, docs/plan/tasks.md, docs/quality-baseline.md, docs/security-baseline.md, CONTRIBUTING.md
- **No tocar:** el resto de docs/ (prd.md, testing-strategy.md, evaluation-plan.md, ADR-0001 … ADR-0006)
- **Hecho cuando:**
  - [ ] Q-3 y Q-13 … Q-24 tienen en su columna Estado la resolución con fecha 2026-11-21, y A-1 y A-11 … A-15 su confirmación.
  - [ ] F-1 … F-12, F-14 … F-16, F-18 y F-19 tienen Estado "Resuelto 2026-11-21" en plan/README.md §3.4; F-13 queda asignado a T-036, que lista el documento que corrige.
  - [ ] Las listas "Tocar solo", dependencias y criterios de las decisiones de ADR-0007 están aplicados a sus tareas, y coinciden la fila de ADR-0007 en plan/README.md §3.3, la columna Decisión de §3.2 y el campo "Decisión" de cada tarea.
  - [ ] Fuera de ADR-0007, open-questions.md y plan/README.md §3.4, ningún documento cita lo sustituido: el nombre antiguo de la prueba de fuentes desactivables (decisión 8) ni la clave primaria antigua de `fact_sources` (decisión 4). El patrón de Verificar escribe una letra entre corchetes para no encontrarse a sí mismo.
  - [ ] Todo enlace relativo a un `.md` de CONTRIBUTING.md y de docs/ apunta a un archivo que existe.
  - [ ] `gitleaks` no encuentra secretos.
- **Verificar:** `test "$(grep -Ec '^\| Q-(3|1[3-9]|2[0-4]) \|.*\| Resuelta 2026-11-21' docs/open-questions.md)" = 13 && test "$(grep -Ec '^\| F-([1-9]|1[0-24-689]) \|.*\| Resuelto 2026-11-21' docs/plan/README.md)" = 17 && ! (grep -rnE 'TestRecall_Sources[T]oggle|primary key \(fact_id, episode[_]id\)' docs CONTRIBUTING.md README.md | grep -Ev '^(docs/adr/ADR-0007[^:]*|docs/open-questions\.md|docs/plan/README\.md):') && test -z "$(for f in CONTRIBUTING.md docs/*.md docs/adr/*.md docs/plan/*.md; do grep -oE '\]\([^)#:]+\.md' "$f" | cut -c3- | while read -r l; do test -f "$(dirname "$f")/$l" || echo "$f: $l"; done; done)" && gitleaks dir . --no-banner`
- **Fuera de alcance:** cualquier código; las preguntas Q-1, Q-2 y Q-5 … Q-12, que siguen abiertas con su responsable.

---

## M0 — Esqueleto andante

### T-001 · Crear el módulo Go, el binario `mnemo serve` y `/healthz`   size:S  risk:LOW
- **Implementa:** FR-25 (parte) · **Decisión:** AD-1, AD-3, ADR-0007
- **Depende de:** supuesto A-1 (ruta del módulo), confirmado en ADR-0007
- **Leer antes:** docs/architecture.md#5-módulos · docs/quality-baseline.md#2-sensores
- **Tocar solo:** go.mod, go.sum, cmd/mnemo/main.go, internal/config/config.go, internal/config/config_test.go, internal/httpapi/server.go, internal/httpapi/health_test.go
- **No tocar:** docs/**, README.md
- **Interfaces:**
  - *Produce:* `config.Load(getenv func(string) string) (config.Config, error)` con campos `DatabaseURL`, `ListenAddr` (por defecto `127.0.0.1:8080`), `ChatBaseURL`, `ChatAPIKey`, `ChatModel`, `EmbedBaseURL`, `EmbedAPIKey`, `EmbedModel`, `EmbedDim`, `RerankURL`; variables `MNEMO_DATABASE_URL`, `MNEMO_LISTEN_ADDR`, `MNEMO_LLM_BASE_URL`, `MNEMO_LLM_API_KEY`, `MNEMO_LLM_MODEL`, `MNEMO_EMBED_BASE_URL`, `MNEMO_EMBED_API_KEY`, `MNEMO_EMBED_MODEL`, `MNEMO_EMBED_DIM`, `MNEMO_RERANK_URL`. `httpapi.NewServer(deps httpapi.Deps) http.Handler`.
- **Hecho cuando:**
  - [ ] `go.mod` declara el módulo y fija `staticcheck` y `govulncheck` con la directiva `tool`.
  - [ ] `mnemo serve` arranca un servidor HTTP con registro JSON por `log/slog` y responde 200 a `GET /healthz`.
  - [ ] Si falta `MNEMO_DATABASE_URL` o `MNEMO_EMBED_DIM` no es 1024, `config.Load` devuelve un error que nombra la variable y el proceso termina con código distinto de cero.
  - [ ] Ningún valor de variable terminada en `_API_KEY` aparece en la salida de registro (prueba con un valor centinela).
- **Verificar:** `test -z "$(gofmt -l .)" && go vet ./... && go tool staticcheck ./... && go test ./internal/config ./internal/httpapi`
- **Fuera de alcance:** base de datos, autenticación, subcomandos distintos de `serve`.
- **Riesgo → mitigación:** la versión de Go instalada no soporta la directiva `tool` → exigir Go ≥ 1.24 en `go.mod`; el fallo es inmediato y explícito.

### T-002 · Levantar PostgreSQL con Docker, aplicar migraciones y exponer `/readyz`   size:M  risk:MED
- **Implementa:** FR-25 (parte) · **Decisión:** ADR-0001, ADR-0003 (roles), ADR-0007
- **Depende de:** T-001
- **Leer antes:** docs/architecture.md#6-modelo-de-datos · docs/adr/ADR-0001-postgresql-unico-almacen.md
- **Tocar solo:** docker-compose.yml, deploy/initdb/01-roles.sql, internal/store/migrate.go, internal/store/migrations/0001_init.sql, internal/store/migrate_test.go, cmd/mnemo/main.go, internal/httpapi/server.go, go.mod, go.sum
- **No tocar:** internal/config/**
- **Interfaces:**
  - *Produce:* `store.Migrate(ctx context.Context, adminURL string) error`; subcomando `mnemo migrate`, que toma la URL de `MNEMO_ADMIN_DATABASE_URL` en `cmd/mnemo` (no en `config.Load`: ADR-0007, decisión 2); servicio `db` de Compose con la imagen `pgvector/pgvector:pg17`, publicado en `127.0.0.1:${MNEMO_DB_PORT:-55432}`, base `mnemo`, rol `mnemo_admin` (dueño, con `BYPASSRLS` y `CREATEDB`: ADR-0007, decisión 2) y rol `mnemo_app` (sin `BYPASSRLS`, sin propiedad, sin `CREATE` en `public`); variables `MNEMO_TEST_DATABASE_URL` y `MNEMO_TEST_ADMIN_DATABASE_URL` documentadas en `docker-compose.yml`.
- **Hecho cuando:**
  - [ ] `0001_init.sql` crea las extensiones `vector` y `pg_trgm` y las tablas `tenants`, `subjects` y `episodes` con las columnas de docs/architecture.md, las de datos (`subjects`, `episodes`) con `tenant_id`, e inserta el inquilino de desarrollo `00000000-0000-0000-0000-000000000001`.
  - [ ] `store.Migrate` aplica cada archivo en una transacción, registra la versión en `schema_migrations` y no hace nada en la segunda ejecución.
  - [ ] `GET /readyz` responde 200 si la base es alcanzable y está en la última versión de migración, y 503 en otro caso.
  - [ ] Si la imagen no trae pgvector ≥ 0.8, la migración falla con un mensaje que lo dice (cierra el supuesto A-9).
- **Verificar:** `docker compose up -d --wait db && go test -tags=integration ./internal/store -run 'TestMigrate_'`
- **Fuera de alcance:** seguridad a nivel de fila (T-012), tablas de hechos (T-010).
- **Riesgo → mitigación:** la etiqueta de la imagen no existe o cambia → fijar la imagen por resumen una vez verificada y anotarlo en docs/open-questions.md Q-8.

### T-003 · Construir la imagen, montar el CI y los comprobadores de arquitectura y dependencias   size:M  risk:LOW
- **Implementa:** FR-25 · **Decisión:** AD-1, SM-C5, docs/security-baseline.md §8, ADR-0007
- **Depende de:** T-002
- **Leer antes:** docs/quality-baseline.md#7-integración-continua · docs/security-baseline.md#8-cadena-de-suministro
- **Tocar solo:** Dockerfile, docker-compose.yml, .github/workflows/ci.yml, tools/depbudget/main.go, tools/archcheck/main.go, .githooks/pre-commit, CONTRIBUTING.md
- **No tocar:** internal/**, cmd/**
- **Interfaces:**
  - *Produce:* `go run ./tools/depbudget -max N` (falla si `go.mod` tiene más de N dependencias directas, sin contar las de la directiva `tool`); `go run ./tools/archcheck` (falla si `internal/resolve`, `internal/retrieve` o `internal/policy` importan `internal/store`, `internal/provider`, `net/http` o `database/sql`, o si `internal/httpapi` o `internal/mcpserver` importan otro paquete `internal/` que no sea `internal/config`; en sus archivos de prueba admite además `internal/store/storetest`, ADR-0007, decisión 3).
- **Hecho cuando:**
  - [ ] `docker compose up -d --wait` levanta `db`, ejecuta el servicio de una vez `admin` (`mnemo migrate`, el único con `MNEMO_ADMIN_DATABASE_URL`) y después `mnemo` (`serve`, que depende de `admin` con `condition: service_completed_successfully` y se conecta como `mnemo_app`), y `/readyz` responde 200 en menos de 60 s (ADR-0007, decisión 2).
  - [ ] El CI falla si, en la salida de `docker compose config`, el `environment` del servicio `mnemo` tiene la variable `MNEMO_ADMIN_DATABASE_URL` o algún valor que contenga `mnemo_admin`.
  - [ ] La imagen se compila en varias etapas, corre con usuario sin privilegios y declara `HEALTHCHECK`.
  - [ ] El flujo de CI ejecuta, en este orden: formato, `go vet`, `staticcheck`, pruebas unitarias, pruebas de integración con un servicio PostgreSQL, `archcheck`, `depbudget -max 6`, `govulncheck`, `go mod verify` y el arranque con Compose; todas las acciones fijadas por SHA y permisos de solo lectura.
  - [ ] `archcheck` falla ante un paquete de prueba que viola la regla (caso negativo incluido en `tools/archcheck`), también cuando la importación prohibida está solo en un archivo `_test.go`.
  - [ ] El servicio PostgreSQL del CI crea los roles con `deploy/initdb/01-roles.sql`, igual que Compose.
  - [ ] `.githooks/pre-commit`, versionado y ejecutable, corre formato, `go vet`, `staticcheck` y `go test ./...` y falla si alguno falla; `CONTRIBUTING.md` explica cómo activarlo con `git config core.hooksPath .githooks` (ADR-0007, decisión 9).
- **Verificar:** `go run ./tools/archcheck && go run ./tools/depbudget -max 6 && docker compose up -d --wait && curl -fsS http://127.0.0.1:8080/readyz`
- **Fuera de alcance:** publicación de imágenes, SBOM (T-036), compuerta de cobertura (T-031).
- **Riesgo → mitigación:** el CI no puede verificarse sin un repositorio remoto → el comando Verificar cubre lo local; la primera ejecución en GitHub se anota como pendiente en docs/open-questions.md hasta verla en verde.

### T-004 · Guardar y recuperar episodios de punta a punta con búsqueda vectorial   size:M  risk:MED
- **Implementa:** FR-1, FR-9 (fuente `episodes.vector`), FR-22 (parte) · **Decisión:** ADR-0001, ADR-0006, ADR-0007
- **Depende de:** T-003
- **Leer antes:** docs/architecture.md#7-api-del-servicio · docs/architecture.md#8-api-pública-del-paquete-go
- **Tocar solo:** mnemo.go, types.go, providers.go, internal/store/tx.go, internal/store/episodes.go, internal/httpapi/episodes.go, internal/httpapi/recall.go, test/e2e/skeleton_test.go, go.mod, go.sum
- **No tocar:** internal/store/migrations/0001_init.sql
- **Interfaces:**
  - *Produce:* `mnemo.Open(ctx, pool, mnemo.Config) (*mnemo.Memory, error)`; `(*Memory).Scope(tenantID [16]byte, subjectID string) *Scope`; `(*Scope).Remember(ctx, []mnemo.Message, ...mnemo.RememberOption) (mnemo.RememberResult, error)`; `(*Scope).Recall(ctx, query string, ...mnemo.RecallOption) (mnemo.RecallResult, error)`; interfaz `mnemo.Embedder { Embed(ctx, []string) ([][]float32, mnemo.Usage, error); Dim() int }`; `mnemo.FakeEmbedder(dim int) mnemo.Embedder`; `store.WithScope(ctx, pool, store.Scope, func(pgx.Tx) error) error`.
- **Hecho cuando:**
  - [ ] `POST /v1/episodes` con `{subject, session_id, messages:[{role, content, occurred_at}]}` guarda un episodio por mensaje con `seq` consecutivo dentro de la sesión, su vector y su `tsvector`, y responde 202 con `episode_ids`.
  - [ ] `POST /v1/recall` con `{subject, query, max_tokens}` devuelve los episodios del sujeto ordenados por distancia coseno, con `score`, y `timings_ms`.
  - [ ] Con `FakeEmbedder`, dos ejecuciones de la misma consulta devuelven exactamente el mismo orden (`TestRecall_Deterministic`).
  - [ ] Dado un sujeto con episodios, cuando otro sujeto del mismo inquilino consulta, entonces no recibe ninguno.
  - [ ] La prueba de punta a punta escribe y lee contra el servicio levantado con Compose.
- **Verificar:** `docker compose up -d --wait --build && go test -tags=integration ./... -run 'TestRecall_Deterministic|TestEpisodes_' && go test -tags=e2e ./test/e2e -run TestSkeleton`
- **Fuera de alcance:** autenticación (el inquilino es el de desarrollo hasta T-012), hechos, fusión, presupuesto de tokens real.
- **Riesgo → mitigación:** la codificación del tipo `vector` con pgx falla → usar `pgvector-go`; si diera problemas, codificar el vector como literal de texto, que PostgreSQL acepta.

### T-005 · Exponer `recall` como herramienta MCP por stdio   size:S  risk:MED
- **Implementa:** FR-21 (parte) · **Decisión:** ADR-0003 (sin parámetro de sujeto), ADR-0007
- **Depende de:** T-004
- **Leer antes:** docs/architecture.md#herramientas-mcp · https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp
- **Tocar solo:** internal/mcpserver/server.go, internal/mcpserver/server_test.go, cmd/mnemo/main.go, go.mod, go.sum
- **No tocar:** internal/httpapi/**, mnemo.go
- **Interfaces:**
  - *Consume:* `(*mnemo.Scope).Recall`.
  - *Produce:* `mcpserver.New(scope *mnemo.Scope) *mcp.Server`; subcomando `mnemo mcp` que lee `MNEMO_MCP_SUBJECT` (sustituido por credencial en T-029).
- **Hecho cuando:**
  - [ ] El servidor registra la herramienta `recall` con `mcp.AddTool` y argumentos `query`, `as_of` (opcional) y `max_tokens`.
  - [ ] El esquema de entrada de la herramienta no contiene las claves `subject` ni `tenant` (aserción explícita).
  - [ ] Una prueba conecta un cliente del SDK al servidor con transporte en memoria, llama a `recall` y recibe el contenido guardado antes con `Remember`.
  - [ ] Queda anotado en docs/open-questions.md qué versión del SDK se fijó y qué revisiones de protocolo declara soportar.
- **Verificar:** `go test -tags=integration ./internal/mcpserver && go run ./tools/depbudget -max 6`
- **Fuera de alcance:** `remember`, `forget`, `timeline`, transporte HTTP (T-029).
- **Riesgo → mitigación:** la API del SDK difiere de lo leído en su documentación → esta tarea existe en M0 justo para descubrirlo pronto; si el SDK no sirve, se anota y T-029 se replantea.

### T-006 · Crear el harness de evaluación mínimo con modelos falsos   size:M  risk:LOW
- **Implementa:** FR-24 (parte) · **Decisión:** ADR-0004, ADR-0007
- **Depende de:** T-004
- **Leer antes:** docs/evaluation-plan.md#4-protocolo · docs/evaluation-plan.md#11-reproducción
- **Tocar solo:** eval/pyproject.toml, eval/uv.lock, eval/src/mnemo_eval/cli.py, eval/src/mnemo_eval/client.py, eval/src/mnemo_eval/pipeline.py, eval/config/smoke.yaml, eval/fixtures/toy.json, eval/tests/test_pipeline.py, .github/workflows/ci.yml, .githooks/pre-commit
- **No tocar:** código Go (internal/**, cmd/**, test/**, *.go)
- **Interfaces:**
  - *Produce:* comando `mnemo-eval run --protocol <yaml> --arms <lista|all> [--split <nombre>]`; `mnemo_eval.client.MnemoClient(base_url, api_key=None)` con `remember(subject, session_id, messages, wait=True)` y `recall(subject, query, **params)`; resultado en `eval/runs/<protocolo>/<brazo>.jsonl` con una línea por pregunta: `question_id, type, answer, correct, reader_input_tokens, recall_ms`.
- **Hecho cuando:**
  - [ ] Con `smoke.yaml` (dos conversaciones de juguete, lector y juez falsos y deterministas) el comando ingiere, consulta, "lee", "juzga" y escribe un archivo por brazo.
  - [ ] El identificador de protocolo es el SHA-256 del archivo YAML y figura en cada línea de resultado.
  - [ ] `pytest` comprueba que una respuesta correcta del lector falso cuenta como acierto y una incorrecta como fallo.
  - [ ] `ruff check` y `ruff format --check` pasan.
  - [ ] `eval/pyproject.toml` declara `pip-audit` como dependencia de desarrollo y registra el marcador `integration` de pytest, de modo que `pytest -m "not integration"` (N2) no avisa de marcadores desconocidos.
  - [ ] El CI ejecuta `uv sync --frozen`, `ruff check`, `ruff format --check`, `pytest`, `pip-audit`, la prueba de punta a punta (`go test -tags=e2e ./test/e2e`) con el servicio levantado por Compose y la evaluación mínima con `smoke.yaml`; `.githooks/pre-commit` añade las líneas de Python (ADR-0007, decisión 9).
- **Verificar:** `docker compose up -d --wait && uv run --project eval ruff check eval && uv run --project eval pytest eval/tests && uv run --project eval mnemo-eval run --protocol eval/config/smoke.yaml --arms all`
- **Fuera de alcance:** conjuntos reales, modelos reales, informe en Markdown (T-009).
- **Riesgo → mitigación:** el harness crece hasta ser un framework → un solo módulo `pipeline.py` con una función por etapa; nada de clases base hasta que haya tres brazos reales.

---

## M1 — Mayor riesgo

### T-007 · Implementar los adaptadores de chat y embeddings compatibles con OpenAI   size:M  risk:HIGH
- **Implementa:** FR-23 (parte) · **Decisión:** ADR-0006, ADR-0007 (decisión 14)
- **Depende de:** T-004
- **Leer antes:** docs/adr/ADR-0006-puertos-de-modelos-y-dimension.md · docs/quality-baseline.md#4-estabilidad
- **Tocar solo:** internal/provider/chat.go, internal/provider/embed.go, internal/provider/retry.go, internal/provider/provider_test.go, providers.go
- **No tocar:** internal/store/**, go.mod
- **Interfaces:**
  - *Produce:* `mnemo.ChatModel { Complete(ctx, mnemo.ChatRequest) (mnemo.ChatResponse, error) }` con `ChatRequest{System, User string; JSONSchema []byte; MaxTokens int}` y `ChatResponse{Text string; Usage mnemo.Usage}`; `mnemo.Usage{InputTokens, OutputTokens int; Provider, Model string}`; `mnemo.OpenAICompatibleChat(baseURL, apiKey, model string) mnemo.ChatModel`; `mnemo.OpenAICompatibleEmbedder(baseURL, apiKey, model string, dim int) mnemo.Embedder`.
- **Hecho cuando:**
  - [ ] Contra un servidor `httptest`: una respuesta 200 se decodifica con su `usage`; un 429 y un 503 se reintentan hasta 3 veces con espera creciente; un 400 no se reintenta; una respuesta que tarda más del tiempo límite (5 s embeddings, 30 s chat, inyectables en la prueba) devuelve error de tiempo agotado.
  - [ ] El embebedor rechaza vectores cuya longitud no sea `dim`.
  - [ ] La clave de API no aparece en ningún mensaje de error.
  - [ ] Una prueba marcada con la etiqueta `live` (no se ejecuta en CI) llama a los endpoints reales configurados por variables de entorno y comprueba salida JSON con esquema y un vector de 1.024 dimensiones; su resultado se anota en docs/open-questions.md (cierra A-4 y A-5).
- **Verificar:** `go test ./internal/provider && go run ./tools/depbudget -max 6`
- **Fuera de alcance:** reranker (T-023), modo por lotes del proveedor, caché de prompts.
- **Riesgo → mitigación:** el endpoint compatible de un proveedor no acepta el esquema JSON → la prueba `live` lo detecta el primer día; alternativa documentada en ADR-0006: adaptador específico.

### T-008 · Descargar y cargar LongMemEval_S y definir el subconjunto de desarrollo   size:S  risk:MED
- **Implementa:** FR-24 (parte) · **Decisión:** ADR-0004
- **Depende de:** T-006
- **Leer antes:** docs/evaluation-plan.md#2-conjuntos-de-datos · https://github.com/xiaowu0162/LongMemEval
- **Tocar solo:** eval/src/mnemo_eval/datasets/longmemeval.py, eval/src/mnemo_eval/cli.py, eval/config/protocol.yaml, eval/tests/test_longmemeval.py
- **No tocar:** eval/src/mnemo_eval/pipeline.py
- **Interfaces:**
  - *Produce:* comando `mnemo-eval fetch`; `load_longmemeval(path) -> list[Question]` con `Question(id, type, question, answer, question_date, sessions: list[Session], evidence_session_ids, is_abstention)`; `split(questions, name) -> list[Question]` para `dev60`, `heldout440`, `write100` y `all`.
- **Hecho cuando:**
  - [ ] `fetch` descarga `longmemeval_s_cleaned.json` y `longmemeval_oracle.json` a `eval/data/`, calcula su SHA-256 y falla si no coincide con el de `protocol.yaml` (la primera vez lo imprime para fijarlo).
  - [ ] El cargador produce 500 preguntas, de las cuales 30 de abstención, e imprime el recuento por tipo.
  - [ ] `dev60` es estratificado por tipo, determinista con la semilla del protocolo, y disjunto de `heldout440`; su unión son las 500.
  - [ ] Se imprime la fracción de sesiones cuyo contenido se repite entre preguntas (mide el supuesto A-6) y se anota en docs/open-questions.md.
- **Verificar:** `uv run --project eval mnemo-eval fetch && uv run --project eval pytest eval/tests/test_longmemeval.py`
- **Fuera de alcance:** LoCoMo (T-033), variante M.
- **Riesgo → mitigación:** el formato del archivo difiere de lo que describe el repositorio → la prueba carga el archivo real y falla con el nombre del campo que falta.

### T-009 · Implementar las líneas base B0, B1 y B2, el lector, el juez y el informe   size:L  risk:HIGH
- **Implementa:** FR-24 · **Decisión:** ADR-0004, ADR-0007 (decisión 14)
- **Depende de:** T-007, T-008
- **Leer antes:** docs/evaluation-plan.md#3-brazos · docs/evaluation-plan.md#5-métricas
- **Tocar solo:** eval/src/mnemo_eval/arms.py, eval/src/mnemo_eval/models.py, eval/src/mnemo_eval/judge.py, eval/src/mnemo_eval/metrics.py, eval/src/mnemo_eval/report.py, eval/src/mnemo_eval/prompts/reader.txt, eval/tests/test_metrics.py, eval/config/protocol.yaml
- **No tocar:** internal/**, eval/src/mnemo_eval/datasets/**
- **Interfaces:**
  - *Consume:* `MnemoClient`, `Question`, `split`.
  - *Produce:* brazos `B0`, `B1`, `B2`, `M` como funciones `arm(question, cfg) -> ArmOutput(context, context_tokens, recall_ms, retrieved_session_ids)`; `judge(question, answer) -> bool` con los prompts por tipo del script oficial; `metrics.accuracy`, `metrics.recall_at_k`, `metrics.mrr`, `metrics.percentile`, `metrics.bootstrap_ci`; comando `mnemo-eval report --out <dir>` que escribe `summary.json` y la tabla en Markdown; comando `mnemo-eval verify-protocol <dir>`.
- **Hecho cuando:**
  - [ ] B0 envía solo la pregunta y su fecha; B1 todo el historial en orden cronológico; B2 ingiere en mnemo y consulta con `sources=["episodes.vector"]` hasta el presupuesto de 8.000 tokens.
  - [ ] Todos los brazos usan el mismo archivo `reader.txt`, temperatura 0, y registran las tokens de entrada que devuelve el proveedor.
  - [ ] Los prompts del juez se copian del script oficial con su procedencia en una cabecera; `test_metrics.py` cubre exactitud por tipo, recall@k excluyendo las preguntas de abstención, percentiles sobre la lista completa y el intervalo por bootstrap con semilla.
  - [ ] `verify-protocol` falla si en un directorio de resultados hay líneas con identificadores de protocolo distintos.
  - [ ] Una ejecución sobre `dev60` de B0 y B2 con modelos reales termina y el informe muestra exactitud, tokens y coste con la tabla de precios fechada.
- **Verificar:** `uv run --project eval pytest eval/tests && uv run --project eval mnemo-eval run --protocol eval/config/smoke.yaml --arms all && uv run --project eval mnemo-eval verify-protocol eval/runs`
- **Fuera de alcance:** validación del juez con etiquetas humanas (T-034), ablaciones (T-032).
- **Riesgo → mitigación:** el juez del artículo no está disponible o es caro → `protocol.yaml` admite otro modelo juez; el cambio queda en el registro de cambios del plan de evaluación y T-034 lo valida.

### T-010 · Extraer hechos (solo añadir) y recuperarlos   size:L  risk:HIGH
- **Implementa:** FR-3, FR-4, FR-9 (fuente `facts.vector`) · **Decisión:** ADR-0002, ADR-0007
- **Depende de:** T-007
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento · docs/adr/ADR-0002-episodios-y-hechos-bitemporales.md
- **Tocar solo:** internal/store/migrations/0002_facts.sql, internal/store/facts.go, internal/extract/extract.go, internal/extract/prompts/extract_v1.txt, internal/extract/extract_test.go, internal/store/facts_test.go, mnemo.go, internal/httpapi/recall.go
- **No tocar:** internal/resolve/** (no existe aún), internal/store/migrations/0001_init.sql
- **Interfaces:**
  - *Produce:* `extract.Extract(ctx, chat mnemo.ChatModel, w extract.Window) ([]extract.Candidate, mnemo.Usage, error)` con `Window{SessionDate time.Time; Context, New []mnemo.Message}` y `Candidate{Subject, Predicate, Object string; ObjectIsEntity bool; Statement string; ValidFrom, ValidTo *time.Time; Confidence float32; SourceSeq []int}`; `store.InsertFact`, `store.SearchFactsByVector`; tablas `facts` y `fact_sources`.
- **Hecho cuando:**
  - [ ] `0002_facts.sql` crea `facts` y `fact_sources` con las restricciones de docs/architecture.md, incluido el índice único parcial sobre `(tenant_id, chain_id) where status = 'active'`, salvo las claves foráneas hacia `entities`, que añade `0007_entities.sql` (A-15, ADR-0007).
  - [ ] `Remember` con `WaitForExtraction` llama al extractor una vez por lote, valida la salida contra el esquema, descarta candidatos sin `SourceSeq` válido e inserta cada hecho como `active` con su propio `chain_id` y sus filas en `fact_sources`, con la instantánea `session_id`, `seq`, `occurred_at` y `role` copiada del episodio (ADR-0007, decisión 4).
  - [ ] Borrar un episodio con hechos deja su fila de `fact_sources` con `episode_id` nulo y la instantánea intacta (prueba de integración `TestFacts_SourceSurvivesEpisodeDelete` en `internal/store/facts_test.go`, que borra el episodio con el rol de migraciones).
  - [ ] Si el candidato no trae `ValidFrom`, se usa `occurred_at` del primer episodio de origen.
  - [ ] Ante JSON inválido se reintenta una vez incluyendo el error; si vuelve a fallar, `Remember` devuelve los episodios guardados y un error de extracción, y los episodios siguen recuperables.
  - [ ] `recall` con `sources` que incluya `facts.vector` devuelve hechos con `statement`, `valid_from`, `status` y `sources`.
  - [ ] Las pruebas usan un `ChatModel` guionizado con tres respuestas grabadas: válida, inválida y luego válida, inválida dos veces.
- **Verificar:** `go test ./internal/extract && go test -tags=integration ./... -run 'TestFR3_|TestFR4_|TestFacts_'`
- **Fuera de alcance:** conflictos y sustitución (T-018, T-019), entidades (T-017), extracción asíncrona (T-015).
- **Riesgo → mitigación:** el prompt extrae demasiado o demasiado poco → no se ajusta aquí; se mide en T-011 sobre `dev60` y se itera con el identificador de versión del prompt.

### T-011 · Ejecutar el experimento M1 sobre 60 preguntas y registrar la decisión   size:M  risk:HIGH
- **Implementa:** FR-24 · **Decisión:** ADR-0002 (disparador de revisión), ADR-0004, ADR-0007 (decisión 14)
- **Depende de:** T-009, T-010
- **Leer antes:** docs/prd.md#dependencies--risks · docs/evaluation-plan.md#10-presupuesto
- **Tocar solo:** eval/src/mnemo_eval/cli.py, eval/src/mnemo_eval/cache.py, eval/tests/test_cache.py, eval/results/m1-dev60/summary.json, docs/open-questions.md, docs/evaluation-plan.md
- **No tocar:** internal/**
- **Interfaces:**
  - *Produce:* comando `mnemo-eval check-m1 <summary.json>` que falla si faltan los brazos B0, B1, B2 o M, si alguno tiene menos de 60 preguntas o si falta el coste por sesión, e imprime la diferencia M − B2 y el coste medido.
- **Hecho cuando:**
  - [ ] La caché de extracción guarda la respuesta por SHA-256 de (contenido de la sesión, identificador del prompt, modelo) y una segunda corrida no hace llamadas de extracción (prueba con un contador).
  - [ ] Existe `eval/results/m1-dev60/summary.json` con los cuatro brazos sobre `dev60`: exactitud global y por tipo, tokens del lector, coste por sesión de ingesta y coste total gastado.
  - [ ] docs/evaluation-plan.md §10 sustituye la estimación por el coste medido, y docs/open-questions.md registra la decisión sobre Q-1 con una de tres salidas: seguir con el plan; seguir con corrida final reducida; pivotar al plan B del riesgo R-1.
  - [ ] Si M − B2 es menor de 3 puntos, la decisión escrita incluye las diez trazas falladas leídas y la hipótesis de por qué.
- **Verificar:** `uv run --project eval pytest eval/tests/test_cache.py && uv run --project eval mnemo-eval check-m1 eval/results/m1-dev60/summary.json`
- **Fuera de alcance:** ajustar el sistema para mejorar el número; eso es M3.
- **Riesgo → mitigación:** gastar el presupuesto iterando aquí → tope de USD 12 para todo M1; al alcanzarlo se decide con lo que haya.

---

## M2 — Cimientos

### T-012 · Aislar inquilinos con seguridad a nivel de fila, credenciales y autenticación   size:L  risk:HIGH
- **Implementa:** FR-19 · **Decisión:** ADR-0003, ADR-0007
- **Depende de:** T-010; T-011 solo en cuanto a la decisión de M1 registrada o aplazada (ADR-0007, decisión 13)
- **Leer antes:** docs/adr/ADR-0003-aislamiento-por-rls.md · docs/adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md · docs/security-baseline.md#4-registro-de-amenazas
- **Tocar solo:** internal/store/migrations/0003_tenancy.sql, internal/store/tx.go, internal/store/keys.go, internal/store/isolation_test.go, internal/httpapi/auth.go, internal/httpapi/auth_test.go, cmd/mnemo/main.go, mnemo.go, eval/src/mnemo_eval/client.py, test/e2e/skeleton_test.go, .github/workflows/ci.yml, CONTRIBUTING.md
- **No tocar:** internal/store/migrations/0001_init.sql, internal/store/migrations/0002_facts.sql
- **Interfaces:**
  - *Produce:* tabla `api_keys`; políticas RLS en todas las tablas con `tenant_id`; función `mnemo_lookup_api_key(p_key_hash bytea)` (`SECURITY DEFINER`, ADR-0007); `store.WithScope` ejecuta `SET LOCAL app.tenant_id`; `store.CreateTenant`, `store.CreateKey(ctx, tenantID, scopes []string, subjectID *string) (plaintext string, err error)`, `store.LookupKey(ctx, plaintext) (store.Principal, error)` con `Principal{TenantID [16]byte; KeyID [16]byte; Scopes []string; SubjectID *string}`, que llama a esa función; `(*mnemo.Memory).Authenticate(ctx, plaintext string) (mnemo.Principal, error)` en el paquete raíz, que es lo que consumen `internal/httpapi` e `internal/mcpserver` (A-14; firma fijada en ADR-0007, decisión 3); subcomandos `mnemo tenant create <nombre>` y `mnemo key create --tenant <id> --scopes <lista> [--subject <id>]`, que se conectan con `MNEMO_ADMIN_DATABASE_URL` (ADR-0007, decisión 2); middleware `httpapi.Authenticate`.
- **Hecho cuando:**
  - [ ] Se escriben primero y se ven fallar: `TestIsolation_AllTables` (para cada tabla con columna `tenant_id` descubierta por `information_schema`: inserta como inquilino A, ejecuta `select * from <tabla>` sin `WHERE` como B y exige 0 filas), `TestRLS_ForcedOnAllTenantTables`, `TestAppRole_NotOwnerNoBypass` (que también exige que `mnemo_app` no tenga `CREATE` en `public` y que `has_table_privilege('mnemo_app', c.oid, 'TRUNCATE')` y `has_table_privilege('mnemo_app', c.oid, 'TRIGGER')` sean falsos para toda tabla de un esquema que no sea del sistema; las migraciones conceden a `mnemo_app`, tabla por tabla, `SELECT, INSERT, UPDATE, DELETE` o menos, nunca `GRANT ALL`), `TestRLS_DefinerFunctionsApproved` (las funciones `SECURITY DEFINER` de todos los esquemas que no son del sistema coinciden, con su firma, con la lista aprobada de ADR-0007; su dueño es `mnemo_admin`; `proconfig` es exactamente `{"search_path=pg_catalog, public, pg_temp"}`; y los roles con `EXECUTE` según `aclexplode(proacl)` son exactamente `mnemo_admin` y `mnemo_app`) y `TestRLS_NoOwnerPrivilegedViews` (`mnemo_app` no tiene ningún privilegio sobre vistas materializadas ni sobre vistas sin `security_invoker = true`, en ningún esquema que no sea del sistema).
  - [ ] Toda ruta `/v1` sin cabecera `Authorization: Bearer` válida responde 401 `UNAUTHENTICATED`; una credencial sin el ámbito de la ruta, 403 `FORBIDDEN`; una credencial ligada al sujeto S que pide el sujeto S', 403.
  - [ ] Dado un hecho del inquilino A, cuando el inquilino B pide `GET /v1/facts/{id}` con ese identificador, entonces recibe 404 `NOT_FOUND`, idéntico al de un identificador inexistente.
  - [ ] La clave tiene el formato `mnemo_<8>_<secreto>`, solo se guarda su SHA-256, y una prueba busca el secreto en claro en todas las columnas de texto de la base y en la salida de registro sin encontrarlo.
  - [ ] Una prueba alterna dos inquilinos sobre una única conexión del pool y nunca ve filas del otro.
  - [ ] El inquilino de desarrollo sin autenticación de T-004 deja de existir como atajo. La prueba de punta a punta crea inquilino y clave. `MnemoClient` toma la credencial de `MNEMO_EVAL_API_KEY` cuando no recibe `api_key`; el CI crea inquilino y clave con `docker compose run --rm -T admin tenant create` y `docker compose run --rm -T admin key create`, en un contenedor efímero y no en el servicio `mnemo`, y pasa la clave en esa variable antes de la evaluación mínima; `CONTRIBUTING.md` explica lo mismo en N4 (ADR-0007, decisión 16).
- **Verificar:** `go test -tags=integration ./internal/store -run 'TestIsolation_|TestRLS_|TestAppRole_' -count=1 && go test -tags=integration ./internal/httpapi -run 'TestAuth_|TestFR19_'`
- **Fuera de alcance:** OAuth, JWT, rotación automática de claves → docs/open-questions.md Q-10.
- **Riesgo → mitigación:** una tabla futura sin política → `TestRLS_ForcedOnAllTenantTables` descubre las tablas dinámicamente y falla sola.

### T-013 · Fijar el contrato OpenAPI, el sobre de error, la validación y los límites   size:M  risk:MED
- **Implementa:** FR-11 (forma de la respuesta), FR-19 (404 uniforme) · **Decisión:** docs/architecture.md §7
- **Depende de:** T-012
- **Leer antes:** docs/architecture.md#7-api-del-servicio · docs/security-baseline.md#7-controles-revisados-contra-asvs-nivel-2
- **Tocar solo:** api/openapi.yaml, internal/httpapi/errors.go, internal/httpapi/validate.go, internal/httpapi/limits.go, internal/httpapi/contract_test.go
- **No tocar:** mnemo.go, internal/store/**
- **Interfaces:**
  - *Produce:* `httpapi.WriteError(w, err)` que traduce `mnemo.ErrNotFound`, `mnemo.ErrConflict`, `mnemo.ErrInvalidState`, `mnemo.ErrForbidden` y errores de validación al sobre `{"error":{"code","message","details"}}`; tabla de rutas `httpapi.Routes` con método, patrón y ámbito, de la que salen el enrutado y la prueba de contrato.
- **Hecho cuando:**
  - [ ] `api/openapi.yaml` describe todas las rutas de docs/architecture.md §7 con sus esquemas, códigos y el sobre de error, antes de implementar las que faltan.
  - [ ] `TestContract_RoutesMatchOpenAPI` falla si una ruta implementada no está en el archivo o al revés (las rutas aún no implementadas se listan en una lista de pendientes dentro de la prueba, que debe quedar vacía al cerrar M3).
  - [ ] Un mensaje de 32.769 bytes responde 413 `PAYLOAD_TOO_LARGE`; 201 mensajes, 422; `max_tokens` 16.001, 422; `occurred_at` más de 24 h en el futuro, 422 `INVALID_OCCURRED_AT`; un cuerpo de más de 1 MB, 413.
  - [ ] Superar el cubo de fichas de la credencial responde 429 `RATE_LIMITED` con `Retry-After`.
  - [ ] Ninguna respuesta de error contiene texto de SQL, rutas de archivo ni trazas (prueba con un error interno provocado).
- **Verificar:** `go test ./internal/httpapi -run 'TestContract_|TestValidate_|TestLimits_|TestErrors_'`
- **Fuera de alcance:** generación de clientes desde OpenAPI.
- **Riesgo → mitigación:** analizar YAML exige una dependencia → el contrato se guarda como JSON válido con extensión `.yaml` (JSON es YAML) y se lee con `encoding/json`; así no entra ninguna dependencia.

### T-014 · Hacer idempotente la escritura de episodios   size:S  risk:MED
- **Implementa:** FR-2 · **Decisión:** docs/architecture.md §6 (consistencia e idempotencia), ADR-0007
- **Depende de:** T-013
- **Leer antes:** docs/architecture.md#consistencia-e-idempotencia
- **Tocar solo:** internal/store/migrations/0004_idempotency.sql, internal/store/idempotency.go, internal/httpapi/episodes.go, internal/httpapi/idempotency_test.go
- **No tocar:** internal/store/episodes.go
- **Hecho cuando:**
  - [ ] `idempotency_keys` lleva `subject_id text not null` (el sujeto del cuerpo), para que la purga la descubra, y su clave primaria es `(tenant_id, subject_id, key)` (ADR-0007, decisión 10).
  - [ ] La misma `Idempotency-Key` usada por dos sujetos distintos del mismo inquilino no produce conflicto entre ellos: cada uno recibe la respuesta de su propia escritura (`TestFR2_KeyScopedBySubject`).
  - [ ] Dada una escritura aceptada con `Idempotency-Key: K`, cuando se repite con K y el mismo cuerpo, entonces la respuesta es byte a byte la original y el número de episodios no cambia.
  - [ ] Con K y otro cuerpo, 409 `IDEMPOTENCY_CONFLICT`.
  - [ ] Sin clave, reenviar los mismos `(session_id, seq)` con el mismo contenido devuelve los identificadores existentes; con otro contenido, 409 `EPISODE_CONFLICT`.
  - [ ] Dos peticiones simultáneas con la misma clave producen un solo conjunto de episodios (prueba con dos gorutinas, 20 repeticiones).
  - [ ] La fila de idempotencia se escribe en la misma transacción que los episodios.
- **Verificar:** `go test -tags=integration ./internal/httpapi -run 'TestFR2_' -count=20`
- **Fuera de alcance:** limpieza de claves caducadas (T-016).
- **Riesgo → mitigación:** comparar cuerpos por texto falla por orden de claves JSON → se compara el SHA-256 del cuerpo canónico (claves ordenadas).

### T-015 · Mover la extracción a una cola de trabajos en PostgreSQL   size:L  risk:HIGH
- **Implementa:** FR-1 (confirmación sin esperar), FR-3 · **Decisión:** ADR-0005, ADR-0007
- **Depende de:** T-014
- **Leer antes:** docs/adr/ADR-0005-extraccion-asincrona-en-postgres.md · docs/quality-baseline.md#4-estabilidad
- **Tocar solo:** internal/store/migrations/0005_jobs.sql, internal/store/jobs.go, internal/jobs/worker.go, internal/jobs/worker_test.go, mnemo.go, internal/httpapi/episodes.go, internal/httpapi/jobs.go, internal/store/isolation_test.go, internal/store/storetest/storetest.go
- **No tocar:** internal/extract/**
- **Interfaces:**
  - *Produce:* `store.EnqueueJob(tx, subjectID, kind, dedupeKey string, payload any) (jobID [16]byte, err error)`; `storetest.IsolatedDB(t testing.TB) (appURL, adminURL string)`, que crea con `mnemo_admin` una base `mnemo_t_<sufijo aleatorio>`, la migra y la borra en `t.Cleanup` (ADR-0007, decisión 2); `store.ClaimJob(ctx) (*store.Job, error)` sobre la función `mnemo_claim_job()` (`SECURITY DEFINER`, `FOR UPDATE SKIP LOCKED` y turno entre inquilinos tal como los define ADR-0007, decisión 2), añadida a la lista de `TestRLS_DefinerFunctionsApproved`; el manejador y el cambio de estado del trabajo corren bajo `store.WithScope` con el inquilino del trabajo; `(*mnemo.Memory).RunWorker(ctx) error`; `jobs.Handler func(ctx, store.Job) (result any, err error)` registrado por `kind`; `GET /v1/jobs/{id}`; parámetro `?wait=true`.
- **Hecho cuando:**
  - [ ] `POST /v1/episodes` guarda los episodios y encola el trabajo `extract` en la misma transacción y responde 202 con `job_id`; con `?wait=true` responde 200 cuando el trabajo termina o 202 si pasan 60 s.
  - [ ] Un trabajo que falla se reintenta con espera exponencial hasta 5 intentos y queda `failed` con el error; los episodios siguen recuperables. El `run_after` del reintento se calcula en SQL (`now() + make_interval(secs => $1)`) y de Go solo llega la duración; la elección entre `queued` y `failed` también la hace SQL (`CASE WHEN attempts >= 5 THEN 'failed' ELSE 'queued' END`). `TestJobs_FailedAfterMaxAttempts`: un trabajo que falla 5 veces acaba `failed`, no `queued` (ADR-0007, decisión 2).
  - [ ] Un trabajo `running` con `locked_at` de más de 5 minutos lo retoma otro ciclo, y el resultado no duplica hechos (prueba que cancela el contexto del primer trabajador a mitad).
  - [ ] El manejador corre con un plazo de 4 minutos, menor que el arriendo. Todo cambio de estado tras el reclamo lleva `WHERE id = $id AND status = 'running' AND attempts = $intento_reclamado`. `TestJobs_LostLeaseIgnored`: cuando el primer trabajador termina después de que otro ciclo retomara el trabajo, su cambio de estado afecta a 0 filas, no devuelve error y no altera el estado de la segunda ejecución (ADR-0007, decisión 2).
  - [ ] `TestJobs_AbandonedNotReclaimed`: un trabajo `running` con `locked_at` de más de 5 minutos y `attempts` = 5 no se retoma y queda `failed` con `error = 'abandoned'` (filas fechadas con el rol de migraciones).
  - [ ] Reenviar el mismo lote no crea un segundo trabajo (clave `session_id:seq_desde-seq_hasta`, única por `(tenant_id, subject_id, kind, dedupe_key)`); dos sujetos del mismo inquilino con el mismo `session_id` y las mismas rondas reciben dos trabajos (`TestJobs_DedupeScopedBySubject`). `jobs.subject_id` es `not null` y el trabajo `extract` lleva `session_id` en `payload` (ADR-0007, decisión 4).
  - [ ] `TestJobs_ClaimTakesTurns`: con diez trabajos listos del inquilino A y uno del B, el de B se reclama a más tardar en el segundo ciclo.
  - [ ] Toda prueba que llama a `mnemo_claim_job()` (directamente, por `store.ClaimJob` o al ejecutar el trabajador) corre en la base de `storetest.IsolatedDB`, no en la de desarrollo (ADR-0007, decisión 2).
  - [ ] `TestRecall_NeverCallsChatModel`: `recall` se ejecuta con un `ChatModel` que hace fallar la prueba si se le llama.
  - [ ] Si el embebedor falla al escribir, el episodio se guarda sin vector y se encola un trabajo `embed`.
- **Verificar:** `go test -tags=integration ./internal/jobs ./internal/httpapi -run 'TestJobs_|TestFR1_|TestRecall_NeverCallsChatModel' -count=5 && go test -tags=integration ./internal/store -run 'TestIsolation_|TestRLS_' -count=1`
- **Fuera de alcance:** varias réplicas del trabajador, prioridades.
- **Riesgo → mitigación:** pruebas inestables por tiempos → el trabajador recibe un reloj y una función de espera inyectables; ninguna prueba usa `time.Sleep`.

### T-016 · Registrar consumo, tiempos por etapa y métricas   size:M  risk:LOW
- **Implementa:** FR-23 · **Decisión:** AD-3, ADR-0006 (decisión 4), ADR-0007
- **Depende de:** T-015
- **Leer antes:** docs/quality-baseline.md#6-observabilidad
- **Tocar solo:** internal/store/migrations/0006_usage.sql, internal/store/usage.go, internal/httpapi/metrics.go, internal/httpapi/usage.go, internal/httpapi/metrics_test.go, mnemo.go, test/e2e/logs_test.go, internal/jobs/cleanup.go, internal/jobs/cleanup_test.go, internal/store/isolation_test.go, internal/store/settings.go, internal/store/settings_test.go, cmd/mnemo/main.go
- **No tocar:** internal/provider/**
- **Interfaces:**
  - *Produce:* interfaz `mnemo.Observer { ProviderCall(op string, u mnemo.Usage, d time.Duration); Stage(op, stage string, d time.Duration) }`; `store.RecordUsage(tx, store.UsageEvent)`; `GET /v1/usage?from=&to=&group_by=operation|model`; `GET /metrics` en formato de texto de Prometheus; limpieza diaria (`internal/jobs/cleanup.go`, tarea periódica del trabajador, no una fila de `jobs`) que llama a `store.Cleanup` y a la función `mnemo_cleanup()` (`SECURITY DEFINER`, ADR-0007, decisión 2), que borra `jobs` terminados de más de 30 días salvo los `extract` en `failed`, `idempotency_keys` de más de 24 h y `usage_events` de más de 180 días; tabla `instance_settings`, `store.EnsureSetting(ctx, pool, key, value string) error` y `store.EnsureEmbedModel(ctx, pool, model string) error`, que la llama con la clave `embed_model` (ADR-0007, decisión 11).
- **Hecho cuando:**
  - [ ] `TestUsage_EveryProviderCallRecorded`: tras un `Remember` con extracción y un `Recall`, hay una fila en `usage_events` por cada llamada hecha a los modelos falsos, con operación, modelo y tokens.
  - [ ] La respuesta de `recall` incluye `usage` y `timings_ms` por etapa.
  - [ ] `/metrics` expone las series de docs/quality-baseline.md §6 y se puede leer sin credencial.
  - [ ] `TestLogs_NoContentLeak`: tras escribir y leer un texto centinela, el centinela no aparece en la salida de registro ni en `/metrics`.
  - [ ] Se registran como eventos, con contador, los fallos de autenticación y las denegaciones por ámbito o sujeto.
  - [ ] `TestCleanup_DeletesOnlyExpired`, en la base de `storetest.IsolatedDB` (ADR-0007, decisión 2): con filas fechadas a ambos lados de cada umbral y de dos inquilinos, la limpieza borra solo las caducadas y conserva los `extract` en `failed` aunque sean antiguos; `mnemo_cleanup()` está en la lista de `TestRLS_DefinerFunctionsApproved`. Las pruebas de esta tarea que ejecutan el trabajador también usan esa base.
  - [ ] `TestSettings_EmbedModelMismatchFails`: `store.EnsureSetting`, con una clave propia de la ejecución (sufijo aleatorio) que la prueba borra al terminar con el rol de migraciones, registra el valor la primera vez, acepta el mismo valor después y devuelve un error que nombra ambos valores si es otro. Usa `INSERT … ON CONFLICT (key) DO NOTHING` seguido de `SELECT`, así que dos arranques simultáneos no fallan. La prueba nunca toca la clave `embed_model` de la base de desarrollo (ADR-0007, decisión 11).
  - [ ] `mnemo serve` y `mnemo mcp` llaman a `store.EnsureEmbedModel` al arrancar y terminan con código distinto de cero si falla (ADR-0006, decisión 4).
- **Verificar:** `docker compose up -d --wait --build && go test -tags=integration ./... -run 'TestUsage_|TestFR23_|TestMetrics_|TestCleanup_|TestSettings_|TestRLS_' && go test -tags=e2e ./test/e2e -run TestLogs_NoContentLeak`
- **Fuera de alcance:** trazas distribuidas, tableros.
- **Riesgo → mitigación:** un histograma propio con errores de cálculo → cubos fijos y una prueba que compara con valores calculados a mano.

---

## M3 — Historias

### T-017 · Resolver entidades y guardar las aristas del grafo   size:M  risk:MED
- **Implementa:** FR-5 · **Decisión:** ADR-0001, ADR-0002, ADR-0007
- **Depende de:** T-015
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento · docs/prd.md#madrigueras-decididas-de-antemano
- **Tocar solo:** internal/store/migrations/0007_entities.sql, internal/store/entities.go, internal/store/entities_test.go, internal/extract/normalize.go, internal/extract/normalize_test.go, internal/jobs/extract_handler.go
- **No tocar:** internal/store/migrations/0002_facts.sql
- **Interfaces:**
  - *Produce:* `extract.NormalizeName(s string) string`; `store.ResolveEntity(tx, name, kind string) (entityID [16]byte, created bool, err error)`; columnas `subject_entity_id` y `object_entity_id` de `facts` pobladas; `0007_entities.sql` añade sus claves foráneas hacia `entities` con `ALTER TABLE` (A-15).
- **Hecho cuando:**
  - [ ] `NormalizeName` pasa a minúsculas, quita acentos, artículos iniciales y espacios repetidos (tabla de casos con nombre, incluidos "La Universidad EAFIT" y "universidad eafit").
  - [ ] Una mención que coincide por nombre normalizado, o con similitud de trigramas ≥ 0,6 dentro del mismo sujeto, se asocia a la entidad existente; en otro caso se crea una.
  - [ ] Dos sujetos distintos con una entidad del mismo nombre tienen dos filas distintas.
  - [ ] Un hecho con `ObjectIsEntity` verdadero queda con `object_entity_id`; uno con valor literal, con `object_value`.
- **Verificar:** `go test ./internal/extract -run TestNormalize && go test -tags=integration ./internal/store -run 'TestFR5_|TestEntities_'`
- **Fuera de alcance:** correferencia con modelo, fusión manual de entidades → docs/open-questions.md Q-11.
- **Riesgo → mitigación:** el umbral 0,6 fusiona entidades distintas ("Ana" y "Anaís") → el umbral es configuración y la taxonomía de fallos de T-035 cuenta estos casos.

### T-018 · Construir el núcleo puro de resolución de conflictos   size:M  risk:HIGH
- **Implementa:** FR-6, FR-7 · **Decisión:** ADR-0002
- **Depende de:** T-010
- **Leer antes:** docs/architecture.md#ciclo-de-vida-de-un-hecho · docs/testing-strategy.md#4-diseño-de-casos
- **Tocar solo:** internal/resolve/resolve.go, internal/resolve/resolve_test.go, internal/resolve/prop_test.go
- **No tocar:** todo lo demás
- **Interfaces:**
  - *Produce:* `resolve.Plan(c resolve.Candidate, existing []resolve.Fact, v resolve.Verdict, now time.Time) ([]resolve.Transition, error)`; `Verdict{Kind resolve.VerdictKind; Targets [][16]byte}` con `VerdictAdd`, `VerdictSupersede`, `VerdictDuplicate`, `VerdictReject`; `Transition{Kind resolve.TransitionKind; FactID [16]byte; SupersededBy *[16]byte; ValidTo *time.Time; NewFact *resolve.Fact}` con `TransitionInsert`, `TransitionSupersede`, `TransitionConfirm`, `TransitionInsertHistorical`; `resolve.CanTransition(from, to resolve.Status) bool`.
- **Hecho cuando:**
  - [ ] Pruebas primero, vistas fallar. Una prueba con nombre por cada transición legal y por cada prohibida del diagrama (`superseded → active`, `retracted → active`, `archived → superseded` devuelven error).
  - [ ] `VerdictSupersede` produce: insertar el nuevo como `active` con el `chain_id` del anterior, y pasar el anterior a `superseded` con `valid_to` igual al `valid_from` del nuevo y `superseded_by` apuntándole.
  - [ ] Si el `valid_from` del candidato es anterior al del hecho activo, el candidato se inserta como histórico (`superseded`, `valid_to` igual al `valid_from` del activo) y el activo no cambia.
  - [ ] `VerdictDuplicate` produce solo una confirmación (actualiza `last_confirmed_at`); ningún hecho nuevo.
  - [ ] Un candidato de confianza menor no puede sustituir a un hecho de confianza mayor (`user` > `assistant` > `untrusted`): `Plan` lo convierte en inserción en cuarentena.
  - [ ] Un veredicto con objetivos que no están en `existing` devuelve error.
  - [ ] Propiedades (`TestProp_`): para cualquier secuencia de candidatos y veredictos, a lo sumo un activo por cadena; el número de hechos nunca disminuye; `valid_to ≥ valid_from`; aplicar dos veces el mismo veredicto duplicado no cambia el estado.
  - [ ] El paquete no importa E/S ni lee el reloj.
- **Verificar:** `go test ./internal/resolve -count=1 -cover && go run ./tools/archcheck`
- **Fuera de alcance:** llamada al modelo y persistencia (T-019).
- **Riesgo → mitigación:** reglas temporales mal entendidas en los bordes → los casos límite `valid_from − 1 ns`, `valid_from`, `valid_to − 1 ns`, `valid_to` son filas de la tabla de pruebas.

### T-019 · Integrar la resolución: veredicto del modelo y transacción   size:L  risk:HIGH
- **Implementa:** FR-6, FR-7 · **Decisión:** ADR-0002, ADR-0005
- **Depende de:** T-017, T-018
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento · docs/security-baseline.md#4-registro-de-amenazas (TH-4)
- **Tocar solo:** internal/extract/verdict.go, internal/extract/prompts/verdict_v1.txt, internal/extract/verdict_test.go, internal/store/facts.go, internal/store/supersede_test.go, internal/jobs/extract_handler.go
- **No tocar:** internal/resolve/**
- **Interfaces:**
  - *Consume:* `resolve.Plan`, `store.ResolveEntity`, `extract.Extract`.
  - *Produce:* `extract.Verdicts(ctx, chat, []extract.Candidate, neighbours map[int][]resolve.Fact) (map[int]resolve.Verdict, mnemo.Usage, error)`; `store.FindNeighbours(tx, c, limit int) ([]resolve.Fact, error)`; `store.ApplyTransitions(tx, []resolve.Transition) error`.
- **Hecho cuando:**
  - [ ] Sin vecinos, el candidato se añade sin llamar al modelo; con enunciado normalizado idéntico a un vecino, se confirma sin llamar al modelo (prueba con un contador de llamadas).
  - [ ] Con vecinos, una sola llamada por lote devuelve un veredicto por candidato, validado contra esquema; un veredicto que nombra un hecho no presentado se descarta y el candidato se añade.
  - [ ] `TestFR6_SupersedeClosesValidity`: el escenario Bogotá → Medellín del PRD, de punta a punta con modelo guionizado.
  - [ ] `TestFR7_DuplicateDoesNotAdd`: el número de hechos activos no cambia.
  - [ ] `TestSupersede_Concurrent`: dos trabajos aplican a la vez sustituciones de la misma cadena; nunca hay dos activos con el mismo `chain_id`; el perdedor reintenta sobre el estado nuevo.
  - [ ] Las transiciones de un lote y el registro de consumo se confirman en una sola transacción.
- **Verificar:** `go test ./internal/extract && go test -tags=integration ./internal/store ./internal/jobs -run 'TestFR6_|TestFR7_|TestSupersede_Concurrent' -count=20`
- **Fuera de alcance:** ajuste del prompt de veredicto (se mide en T-032).
- **Riesgo → mitigación:** el modelo sustituye en exceso → el veredicto solo ve vecinos con la misma entidad-sujeto; la tasa de sustituciones por sesión se imprime en el informe de evaluación.

### T-020 · Consultar con fecha de referencia y devolver la línea de tiempo   size:M  risk:MED
- **Implementa:** FR-8 · **Decisión:** ADR-0002, ADR-0007
- **Depende de:** T-019
- **Leer antes:** docs/architecture.md#7-api-del-servicio · docs/prd.md#criterios-de-aceptación-flujo-principal-y-errores
- **Tocar solo:** internal/retrieve/asof.go, internal/retrieve/asof_test.go, internal/store/facts.go, internal/httpapi/timeline.go, internal/httpapi/facts.go, mnemo.go, options.go
- **No tocar:** internal/resolve/**
- **Interfaces:**
  - *Produce:* `retrieve.FilterAsOf(facts []retrieve.Fact, asOf time.Time, includeHistory bool) []retrieve.Fact`; opciones `mnemo.AsOf(t)` e `mnemo.IncludeHistory()`; `(*Scope).Timeline(ctx, entity string) ([]mnemo.Fact, error)`; `(*Scope).Facts(ctx, mnemo.FactFilter) (mnemo.FactPage, error)`; rutas `GET /v1/subjects/{subject}/timeline`, `GET /v1/subjects/{subject}/facts`, `GET /v1/facts/{id}`.
- **Hecho cuando:**
  - [ ] Propiedad: un hecho pasa `FilterAsOf` si y solo si `valid_from ≤ asOf < valid_to`, tratando `valid_from` nulo como menos infinito y `valid_to` nulo como más infinito.
  - [ ] `TestFR8_AsOfReturnsFactValidThen`: con el estado Bogotá → Medellín, `as_of` 2026-05-01 devuelve primero Bogotá; sin `as_of`, Medellín.
  - [ ] `timeline` devuelve la cadena de hechos de la entidad ordenada por `valid_from`, incluidos los sustituidos, con sus intervalos.
  - [ ] El listado de hechos pagina por cursor con máximo de 200 y filtra por `status`, `entity` y `as_of`.
- **Verificar:** `go test ./internal/retrieve -run 'TestAsOf|TestProp_AsOf' && go test -tags=integration ./... -run 'TestFR8_|TestTimeline_|TestFactsPage_'`
- **Fuera de alcance:** consultas por tiempo de sistema (`recorded_at`) → docs/open-questions.md Q-12.
- **Riesgo → mitigación:** zonas horarias → todo se guarda y compara en UTC; las entradas sin zona se rechazan con 422.

### T-021 · Añadir la fuente léxica y la fusión por rango recíproco   size:M  risk:MED
- **Implementa:** FR-9 · **Decisión:** ADR-0001 (texto completo nativo), característica rectora, ADR-0007
- **Depende de:** T-010
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento · docs/quality-baseline.md#1-funciones-de-aptitud
- **Tocar solo:** internal/retrieve/fuse.go, internal/retrieve/fuse_test.go, internal/store/search.go, internal/store/search_test.go, mnemo.go, options.go
- **No tocar:** internal/httpapi/** (el parámetro `sources` ya pasa tal cual)
- **Interfaces:**
  - *Produce:* `retrieve.FuseRRF(lists [][]retrieve.ID, k int) []retrieve.Scored`; `store.SearchEpisodesFTS`, `store.SearchFactsFTS`, `store.SearchEpisodesByVector`, `store.SearchFactsByVector`, todas con límite 40 y filtradas por `(tenant_id, subject_id)`; tipo `mnemo.Source` con constantes `SourceFactsVector`, `SourceFactsFTS`, `SourceFactsGraph`, `SourceEpisodesVector`, `SourceEpisodesFTS`; opción `mnemo.Sources(...mnemo.Source)`.
- **Hecho cuando:**
  - [ ] `FuseRRF` calcula `Σ 1/(k + rango)` con k = 60; propiedades: invariante al orden de las listas; un documento presente en más listas nunca puntúa menos que el mismo en un subconjunto de ellas; los empates se resuelven por identificador, de forma determinista.
  - [ ] Las búsquedas léxicas usan `websearch_to_tsquery` con la configuración de texto completo del inquilino y `ts_rank_cd`.
  - [ ] `TestFR9_SourcesToggle`: con cada combinación de fuentes, la respuesta solo contiene elementos de las fuentes pedidas e informa cuáles se usaron.
  - [ ] Una consulta con un nombre propio poco frecuente que la fuente vectorial falsa no encuentra sí aparece con la fuente léxica.
  - [ ] Una prueba inspecciona el SQL de `internal/store/search.go` y falla si alguna consulta vectorial no filtra por `tenant_id` y `subject_id`.
- **Verificar:** `go test ./internal/retrieve -run 'TestFuse|TestProp_Fuse' -count=1 && go test -tags=integration ./... -run 'TestFR9_SourcesToggle|TestSearch_'`
- **Fuera de alcance:** BM25 real (deuda D-1), índice ANN (deuda D-2).
- **Riesgo → mitigación:** `ts_rank_cd` ordena mal textos cortos → la fusión usa solo rangos, no puntuaciones, así que la escala no importa; la calidad se mide en la ablación A2.

### T-022 · Añadir la fuente de grafo   size:M  risk:MED
- **Implementa:** FR-9 (fuente `facts.graph`), FR-5 · **Decisión:** ADR-0001
- **Depende de:** T-017, T-021
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento
- **Tocar solo:** internal/store/graph.go, internal/store/graph_test.go, mnemo.go
- **No tocar:** internal/retrieve/fuse.go
- **Interfaces:**
  - *Produce:* `store.MatchEntities(tx, query string, limit int) ([][16]byte, error)`; `store.SearchFactsByGraph(tx, entityIDs [][16]byte, hops int, limit int) ([]retrieve.ID, error)`.
- **Hecho cuando:**
  - [ ] Las entidades de la consulta se detectan por coincidencia de nombre normalizado, trigramas o texto completo contra `entities`, sin llamar a ningún modelo.
  - [ ] Dada la entidad "Laura" con los hechos "Laura trabaja en Acme" y "Acme está en Cali", la consulta "¿dónde trabaja Laura?" devuelve por la fuente de grafo ambos hechos: el primero directo y el segundo a un salto.
  - [ ] El recorrido se limita a un salto adicional y a 40 hechos, ordenados por `last_confirmed_at` descendente.
  - [ ] Con `sources` sin `facts.graph`, no se ejecuta ninguna consulta contra `entities` (prueba con contador de consultas).
- **Verificar:** `go test -tags=integration ./internal/store -run 'TestGraph_' && go test -tags=integration ./... -run TestFR9_SourcesToggle`
- **Fuera de alcance:** más de dos saltos, pesos por tipo de relación.
- **Riesgo → mitigación:** la fuente de grafo no aporta exactitud → es una ablación (A5); si no aporta, se dice en los resultados y la fuente queda desactivada por defecto.

### T-023 · Añadir el reranker con degradación   size:S  risk:LOW
- **Implementa:** FR-10 · **Decisión:** ADR-0006, ADR-0007 (decisión 14)
- **Depende de:** T-021
- **Leer antes:** docs/quality-baseline.md#4-estabilidad · docs/adr/ADR-0006-puertos-de-modelos-y-dimension.md
- **Tocar solo:** internal/provider/rerank.go, internal/provider/rerank_test.go, providers.go, mnemo.go, options.go
- **No tocar:** internal/retrieve/**
- **Interfaces:**
  - *Produce:* `mnemo.Reranker { Rerank(ctx, query string, docs []string) ([]float64, mnemo.Usage, error) }`; `mnemo.HTTPReranker(url, apiKey string) mnemo.Reranker`; opción `mnemo.WithRerank()`; campo `Degraded []string` en `mnemo.RecallResult`.
- **Hecho cuando:**
  - [ ] Con reranker configurado y pedido, los 50 primeros candidatos fusionados se reordenan por la puntuación devuelta.
  - [ ] `TestFR10_RerankTimeoutDegrades`: un reranker que tarda más de 1,5 s (tiempo inyectable) deja el orden fusionado y `degraded` contiene `"rerank"`; la respuesta no es un error.
  - [ ] Tras 5 fallos seguidos el cortacircuitos se abre 30 s y no se hacen llamadas (prueba con reloj inyectado).
  - [ ] Sin reranker configurado, pedirlo no falla y `degraded` contiene `"rerank"`.
  - [ ] El formato real de respuesta del reranker elegido se comprueba con una prueba de etiqueta `live` y se anota en docs/open-questions.md.
- **Verificar:** `go test ./internal/provider -run 'TestRerank' && go test -tags=integration ./... -run 'TestFR10_'`
- **Fuera de alcance:** reranker local, reordenación por modelo generativo.
- **Riesgo → mitigación:** el reranker empeora los resultados → ablación A6; desactivado por defecto.

### T-024 · Ensamblar el contexto dentro del presupuesto e indicar ausencia de evidencia   size:M  risk:MED
- **Implementa:** FR-11, FR-12 · **Decisión:** AD-4, ADR-0007
- **Depende de:** T-020, T-021
- **Leer antes:** docs/architecture.md#7-api-del-servicio · docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento
- **Tocar solo:** internal/retrieve/assemble.go, internal/retrieve/assemble_test.go, mnemo.go, options.go, internal/httpapi/recall.go, internal/httpapi/recall_test.go
- **No tocar:** internal/store/**
- **Interfaces:**
  - *Produce:* `retrieve.Assemble(facts []retrieve.ScoredFact, episodes []retrieve.ScoredEpisode, maxTokens int) retrieve.Context` con `Context{Text string; FactIDs, EpisodeIDs []retrieve.ID; EstimatedTokens int}`; `retrieve.EstimateTokens(s string) int` (caracteres / 4, redondeo hacia arriba); opción `mnemo.MaxTokens(n)`; campo `NoEvidence bool`.
- **Hecho cuando:**
  - [ ] El texto empieza por `<memory>` y termina por `</memory>`; cada hecho ocupa una línea con su intervalo de validez, su nivel de confianza y su enunciado; cada episodio, una línea con fecha, rol y contenido.
  - [ ] Orden: hechos por puntuación, luego la ronda de origen del primer hecho, luego episodios por puntuación; ningún elemento se corta a la mitad; `EstimatedTokens ≤ maxTokens` siempre (propiedad).
  - [ ] `TestFR11_ProvenanceOnEveryItem` (integración, en `internal/httpapi/recall_test.go`): cada hecho devuelto trae al menos una procedencia con `session_id`, `seq`, `occurred_at` y `role` (la instantánea de `fact_sources`), y `episode_id` cuando el episodio sigue existiendo; también con `episode_id` nulo, tras borrar el episodio con el rol de migraciones (ADR-0007, decisión 4). El caso tras resumir lo cubre T-026.
  - [ ] `TestFR12_NoEvidence`: sin coincidencia léxica ni de grafo y con similitud máxima bajo el umbral (0,25, configurable), `no_evidence` es verdadero y el texto es `<memory></memory>`.
  - [ ] Un sujeto inexistente devuelve respuesta vacía con `no_evidence`, no un error.
- **Verificar:** `go test ./internal/retrieve -run 'TestAssemble|TestProp_Assemble' -count=1 && go test -tags=integration ./... -run 'TestFR11_|TestFR12_'`
- **Fuera de alcance:** tokenizador exacto (deuda D-3), resumen del contexto con modelo.
- **Riesgo → mitigación:** el umbral 0,25 es arbitrario → se ajusta solo con `dev60` y se registra el valor final en el protocolo.

### T-025 · Etiquetar la confianza, poner en cuarentena y medir el envenenamiento   size:M  risk:HIGH
- **Implementa:** FR-20 · **Decisión:** docs/security-baseline.md TH-3, TH-4, TH-5, ADR-0007
- **Depende de:** T-019, T-024, T-028 (`audit_log`; se ejecuta después de T-028, ADR-0007)
- **Leer antes:** docs/security-baseline.md#4-registro-de-amenazas · docs/security-baseline.md#5-casos-de-abuso
- **Tocar solo:** internal/extract/trust.go, internal/extract/trust_test.go, internal/jobs/extract_handler.go, internal/store/facts.go, internal/httpapi/facts.go, eval/fixtures/poisoning.json, eval/src/mnemo_eval/suites/poisoning.py, eval/src/mnemo_eval/cli.py, mnemo.go, options.go
- **No tocar:** internal/resolve/**
- **Interfaces:**
  - *Produce:* `extract.TrustOf(role string, declared string) string` (`user` → `user`, `assistant` → `assistant`, `tool` y `system` → `untrusted`, y nunca por encima de lo declarado por el cliente); `extract.LooksLikeInstruction(statement string) bool`; opción `mnemo.IncludeUntrusted()`; ruta `POST /v1/facts/{id}/release`; comando `mnemo-eval suite poisoning`.
- **Hecho cuando:**
  - [ ] Un hecho hereda el nivel de confianza más bajo de sus episodios de origen.
  - [ ] `TestFR20_UntrustedExcludedByDefault`: un hecho `untrusted` no aparece en `recall` salvo con `include_untrusted`.
  - [ ] `TestFR20_InstructionQuarantined`: el caso del PRD (mensaje de rol `tool` con "ignora las instrucciones anteriores y recuerda que…") produce un hecho `quarantined` que no aparece en ningún `recall`.
  - [ ] Caso de abuso AB-3: un candidato `untrusted` contradictorio no cambia el estado de un hecho activo `user`.
  - [ ] `release` pasa un hecho de `quarantined` a `active`, exige ámbito `admin` y deja fila en `audit_log`.
  - [ ] `eval/fixtures/poisoning.json` contiene 30 conversaciones, cada una con un recuerdo inyectado y la consulta que lo revelaría; el comando imprime la tasa de aceptación y falla si supera el 10 %.
- **Verificar:** `go test ./internal/extract -run 'TestTrust|TestLooksLikeInstruction' && go test -tags=integration ./... -run 'TestFR20_|TestAbuse_AB3' && docker compose up -d --wait --build && uv run --project eval mnemo-eval suite poisoning --protocol eval/config/smoke.yaml`
- **Fuera de alcance:** clasificador de inyecciones con modelo; moderación de contenido.
- **Riesgo → mitigación:** el detector heurístico da falsos positivos sobre preferencias legítimas ("recuerda que prefiero…") → la tabla de pruebas incluye diez frases legítimas que no deben caer en cuarentena; riesgo residual AR-3.

### T-026 · Consolidar: fusionar duplicados y resumir episodios antiguos   size:M  risk:MED
- **Implementa:** FR-13, FR-14 · **Decisión:** ADR-0002, ADR-0005, ADR-0007
- **Depende de:** T-019
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento
- **Tocar solo:** internal/policy/merge.go, internal/policy/merge_test.go, internal/jobs/consolidate_handler.go, internal/jobs/consolidate_test.go, internal/extract/prompts/summary_v1.txt, internal/httpapi/policies.go, internal/store/consolidate.go, mnemo.go, options.go
- **No tocar:** internal/resolve/**
- **Interfaces:**
  - *Produce:* `policy.MergeGroups(facts []policy.FactVec, threshold float64) [][]policy.ID` (agrupa hechos activos de la misma entidad-sujeto con coseno ≥ umbral); trabajo `consolidate`; `(*Scope).Consolidate(ctx) (mnemo.JobRef, error)`; ruta `POST /v1/subjects/{subject}/consolidate`; clave de política `episode_retention_days` en `subjects.policy`.
- **Hecho cuando:**
  - [ ] `TestFR13_MergeKeepsProvenance`: de un grupo de duplicados queda activo el más reciente, con la unión de las procedencias; los demás pasan a `archived` con `superseded_by` apuntándole.
  - [ ] `MergeGroups` es determinista y no agrupa hechos de entidades-sujeto distintas (propiedad).
  - [ ] `TestFR14_SummarizeOnlyAfterExtraction`: con `episode_retention_days` definido, los episodios de una sesión más antigua que la ventana se sustituyen por un episodio de rol `summary` solo si ningún trabajo `extract` de ese sujeto y esa sesión (filtro por `tenant_id`, `subject_id` y `payload->>'session_id'`) está `queued`, `running` o `failed`; si no, no se toca nada. Casos de la prueba: trabajo `done`, trabajo `done` ya borrado por la limpieza (se resume), trabajo `failed` (no se resume), y otro sujeto con el mismo `session_id` y su `extract` en `queued` (no impide resumir el del primero) (ADR-0007, decisión 4). Antes de borrar los originales, y en la misma transacción, las filas de `fact_sources` que los citan se reapuntan al episodio `summary` sin tocar su instantánea; tras resumir, cada hecho de la sesión sigue devolviendo en `recall` su procedencia con `session_id`, `seq`, `occurred_at` y `role` de la ronda original (FR-11, ADR-0007, decisión 4).
  - [ ] Sin `episode_retention_days`, ningún episodio se borra.
  - [ ] Pedir la consolidación dos veces el mismo día devuelve el mismo trabajo.
- **Verificar:** `go test ./internal/policy -run 'TestMerge|TestProp_Merge' && go test -tags=integration ./internal/jobs -run 'TestFR13_|TestFR14_'`
- **Fuera de alcance:** perfil resumido del sujeto, consolidación programada (se lanza por API).
- **Riesgo → mitigación:** el resumen pierde la evidencia que una pregunta necesitaba → desactivado por defecto; su efecto se mide en la ablación A7.

### T-027 · Olvidar por decaimiento   size:M  risk:MED
- **Implementa:** FR-15 · **Decisión:** ADR-0002, ADR-0007
- **Depende de:** T-026
- **Leer antes:** docs/architecture.md#9-algoritmos-que-fijan-el-comportamiento
- **Tocar solo:** internal/policy/decay.go, internal/policy/decay_test.go, internal/jobs/forget_handler.go, internal/jobs/forget_test.go, internal/store/facts.go, internal/httpapi/policies.go, mnemo.go, options.go
- **No tocar:** internal/resolve/**
- **Interfaces:**
  - *Produce:* `policy.DecayScore(s policy.FactStats, now time.Time, p policy.Policy) float64` con `FactStats{Importance float64; LastConfirmedAt time.Time; AccessCount int}` y `Policy{HalfLifeDays, Threshold float64; MinAgeDays int}`; trabajo `forget`; `(*Scope).Forget(ctx) (mnemo.JobRef, error)`; ruta `POST /v1/subjects/{subject}/forget`; actualización por lotes de `access_count` y `last_accessed_at`, desactivable con `mnemo.WithoutAccessTracking()`.
- **Hecho cuando:**
  - [ ] `DecayScore = importance × 0,5^(edad_días / HalfLifeDays) × (1 + ln(1 + access_count))`; propiedades: decrece con la edad, crece con los accesos, es 0 si la importancia es 0.
  - [ ] `TestFR15_ArchivedExcludedAndRestorable`: un hecho con puntuación bajo el umbral y más antiguo que `MinAgeDays` pasa a `archived`, deja de aparecer en `recall` y vuelve a `active` cuando llega un candidato que lo confirma.
  - [ ] Los valores por defecto son 90 días, 0,05 y 30 días, y se sustituyen por los de `subjects.policy`.
  - [ ] Con `track_access=false`, `recall` no modifica ninguna fila (prueba que compara un volcado de `facts` antes y después).
- **Verificar:** `go test ./internal/policy -run 'TestDecay|TestProp_Decay' -count=1 && go test -tags=integration ./internal/jobs ./... -run 'TestFR15_|TestRecall_NoWritesWithoutTracking'`
- **Fuera de alcance:** olvido de episodios (lo cubre la retención de T-026), olvido programado.
- **Riesgo → mitigación:** archivar lo que luego hace falta → archivar es reversible y la curva exactitud frente a umbral es la ablación A8.

### T-028 · Purgar, retractar y exportar   size:L  risk:HIGH
- **Implementa:** FR-16, FR-17, FR-18 · **Decisión:** ADR-0003, ADR-0007, docs/security-baseline.md TH-13, TH-14, TH-18
- **Depende de:** T-016, T-019
- **Leer antes:** docs/security-baseline.md#6-datos-personales-y-supresión · docs/architecture.md#consistencia-e-idempotencia
- **Tocar solo:** internal/store/migrations/0008_audit.sql, internal/store/purge.go, internal/store/purge_test.go, internal/store/export.go, internal/store/audit.go, internal/httpapi/subjects.go, internal/httpapi/subjects_test.go, mnemo.go
- **No tocar:** internal/store/migrations/0001_init.sql … 0007_entities.sql
- **Interfaces:**
  - *Produce:* `(*Scope).Purge(ctx) (mnemo.DeletionReceipt, error)` con `DeletionReceipt{SubjectRef [32]byte; Deleted map[string]int; At time.Time}`; `(*Scope).Retract(ctx, factID [16]byte, reason string) error`; `(*Scope).Export(ctx, w io.Writer) error`; rutas `DELETE /v1/subjects/{subject}`, `POST /v1/facts/{id}/retract`, `GET /v1/subjects/{subject}/export`; tabla `audit_log` con solo `INSERT` para `mnemo_app`.
- **Hecho cuando:**
  - [ ] Prueba primero: `TestPurge_NoRowsLeftAnyTable` descubre por `information_schema` todas las tablas con columna `subject_id`, purga un sujeto con datos en todas y exige 0 filas suyas en cada una (salvo `usage_events`, donde `subject_id` debe quedar nulo). La lista incluye `idempotency_keys` y `api_keys`: la purga borra las credenciales ligadas al sujeto (ADR-0007, decisión 10).
  - [ ] `TestFR16_ReceiptMatchesCounts`: los recuentos del recibo coinciden con los previos, incluidos `api_keys` e `idempotency_keys`; con una credencial del inquilino no ligada a sujeto, una segunda purga devuelve ceros y el `recall` posterior devuelve vacío con `no_evidence`; con la credencial ligada al sujeto purgado, ambas peticiones reciben 401 `UNAUTHENTICATED` (ADR-0007, decisión 10).
  - [ ] Un trabajo de extracción que termina después de la purga no escribe nada (falla por `subjects.purged_at`).
  - [ ] `TestFR17_RetractExcludesAndAudits`: el hecho pasa a `retracted`, no aparece en `recall` y hay una fila en `audit_log` con el motivo.
  - [ ] `TestFR18_ExportRoundTrip`: la exportación es NDJSON con una línea por episodio, hecho (con su cadena) y entidad, y contiene todos los identificadores del sujeto y ninguno de otro.
  - [ ] `UPDATE` y `DELETE` sobre `audit_log` fallan con el rol de la aplicación; la fila de auditoría de la purga guarda `sha256(tenant_id ‖ subject_id)` y no el identificador.
  - [ ] Una credencial ligada al sujeto S no puede purgar ni exportar S'.
- **Verificar:** `go test -tags=integration ./internal/store ./internal/httpapi -run 'TestPurge_|TestFR16_|TestFR17_|TestFR18_|TestAudit_' -count=1`
- **Fuera de alcance:** borrado en copias de seguridad (del operador), importación.
- **Riesgo → mitigación:** una tabla futura con datos del sujeto queda fuera de la purga → la prueba descubre las tablas dinámicamente y la purga itera sobre la misma lista.

### T-029 · Completar el servidor MCP y añadir el transporte HTTP   size:M  risk:MED
- **Implementa:** FR-21 · **Decisión:** ADR-0003, docs/security-baseline.md TH-2, TH-19, ADR-0007
- **Depende de:** T-012, T-020, T-028
- **Leer antes:** docs/architecture.md#herramientas-mcp · docs/security-baseline.md#análisis-de-flujo-tóxico
- **Tocar solo:** internal/mcpserver/server.go, internal/mcpserver/tools.go, internal/mcpserver/server_test.go, internal/mcpserver/testdata/tools.golden.json, internal/httpapi/server.go, cmd/mnemo/main.go
- **No tocar:** mnemo.go, internal/store/**
- **Interfaces:**
  - *Consume:* `(*Scope).Remember`, `Recall`, `Timeline`, `Retract`, `Purge`; `(*Memory).Authenticate` (T-012; `internal/mcpserver` no importa `internal/store`, A-14).
  - *Produce:* herramientas `remember`, `recall`, `timeline`, `forget`; `mnemo mcp` autenticado con `MNEMO_API_KEY`; ruta `/mcp` con transporte Streamable HTTP y la misma cabecera `Bearer`.
- **Hecho cuando:**
  - [ ] Las cuatro herramientas existen con los argumentos y anotaciones de docs/architecture.md; sus esquemas se comparan con `tools.golden.json` y ninguna declara `subject` ni `tenant`.
  - [ ] `TestFR21_SubjectBoundByCredential`: con una credencial ligada a S, ninguna combinación de argumentos devuelve memoria de S'; con una credencial no ligada a sujeto, el servidor MCP se niega a arrancar.
  - [ ] `forget` con `everything: true` y `confirm` distinto del identificador del sujeto devuelve un error que explica qué falta, sin borrar nada.
  - [ ] Los errores de herramienta son mensajes accionables ("`max_tokens` debe estar entre 1 y 4000"), no trazas.
  - [ ] `recall` sin evidencia devuelve un texto que lo dice, no una cadena vacía.
  - [ ] Por HTTP: una petición sin credencial recibe 401; una con cabecera `Origin` no permitida, 403.
  - [ ] Queda anotado en docs/open-questions.md qué revisión del protocolo se negoció en la prueba con el cliente del SDK.
- **Verificar:** `go test -tags=integration ./internal/mcpserver -run 'TestFR21_|TestTools_|TestMCPHTTP_' && go run ./tools/archcheck`
- **Fuera de alcance:** OAuth para MCP (riesgo aceptado AR-2), recursos y prompts de MCP.
- **Riesgo → mitigación:** el modo sin estado del transporte HTTP no se configura como se esperaba → si no se logra en dos intentos, se entrega solo stdio (está en el orden de recorte) y se anota.

### T-030 · Cerrar la API pública del paquete Go con ejemplos ejecutables   size:M  risk:LOW
- **Implementa:** FR-22 · **Decisión:** docs/architecture.md §8, ADR-0007
- **Depende de:** T-024, T-027, T-028
- **Leer antes:** docs/architecture.md#8-api-pública-del-paquete-go
- **Tocar solo:** mnemo.go, types.go, options.go, errors.go, example_test.go, doc.go, docs/architecture.md
- **No tocar:** internal/**
- **Hecho cuando:**
  - [ ] Los identificadores exportados del paquete raíz coinciden con docs/architecture.md §8, incluido `mnemo.Migrate(ctx, adminURL string) error`, que envuelve `store.Migrate` (ADR-0007, decisión 17); lo que sobre se quita o se deja de exportar, y lo que cambie se actualiza en el documento en el mismo commit.
  - [ ] `example_test.go` contiene `ExampleScope_Remember`, `ExampleScope_Recall` y `ExampleScope_Timeline`, que se ejecutan contra PostgreSQL con `FakeEmbedder` y un `ChatModel` guionizado y comprueban su salida.
  - [ ] `TestFR22_LibraryWithoutHTTP`: todas las operaciones se ejercitan desde el paquete sin levantar el servidor, y el binario de esa prueba no enlaza `internal/httpapi` ni el SDK de MCP (comprobado con `go list -deps`).
  - [ ] Con `Config.Chat` nulo, `Remember` guarda episodios y no encola extracción.
  - [ ] Cada función exportada tiene comentario de documentación; `go vet` no avisa.
- **Verificar:** `go vet . && go test -tags=integration . -run 'Example|TestFR22_' && go run ./tools/archcheck`
- **Fuera de alcance:** SDK de TypeScript, cliente Python publicado en un índice de paquetes.
- **Riesgo → mitigación:** romper a quien ya use el paquete → no hay usuarios antes de la versión 0.1.0; después, versionado semántico.

### T-031 · Añadir la compuerta de cobertura y la trazabilidad de requisitos   size:S  risk:LOW
- **Implementa:** docs/testing-strategy.md §5 y §6 (cubre FR-1 … FR-23 por nombre de prueba) · **Decisión:** ADR-0001 (lista negra), ADR-0007
- **Depende de:** T-030
- **Leer antes:** docs/testing-strategy.md#5-cobertura · docs/testing-strategy.md#6-trazabilidad
- **Tocar solo:** tools/covergate/main.go, tools/covergate/main_test.go, tools/fr-trace/main.go, .github/workflows/ci.yml, internal/httpapi/contract_test.go, tools/depbudget/main.go, tools/depbudget/main_test.go
- **No tocar:** internal/** salvo el archivo indicado
- **Interfaces:**
  - *Produce:* `go run ./tools/covergate -profile coverage.out` (umbrales por paquete de docs/testing-strategy.md §1; falla si alguno queda por debajo); `go run ./tools/fr-trace -from 1 -to 23` (falla si para algún requisito no existe una prueba cuyo nombre empiece por `TestFR<n>_`); `go run ./tools/depbudget -max 6` falla además si `go.mod` requiere un cliente de otra base de datos, cola o caché (lista negra de ADR-0007, decisión 11).
- **Hecho cuando:**
  - [ ] `depbudget` falla con un `go.mod` de prueba que requiere un módulo de la lista negra y pasa sin él (casos en `tools/depbudget/main_test.go`).
  - [ ] `covergate` falla con un perfil de prueba por debajo del umbral y pasa con uno por encima (casos en `main_test.go`).
  - [ ] `fr-trace` lista la prueba de cada requisito de FR-1 a FR-23 y no falta ninguna.
  - [ ] La lista de rutas pendientes de `TestContract_RoutesMatchOpenAPI` está vacía.
  - [ ] El CI ejecuta ambos programas.
- **Verificar:** `go test ./tools/... && go test -tags=integration -coverprofile=coverage.out ./... && go run ./tools/covergate -profile coverage.out && go run ./tools/fr-trace -from 1 -to 23`
- **Fuera de alcance:** pruebas de mutación (docs/open-questions.md Q-9).
- **Riesgo → mitigación:** perseguir el porcentaje con pruebas vacías → la autorrevisión de CONTRIBUTING.md pregunta si la prueba pasa al borrar la aserción.

---

## M4 — Evaluación y publicación

### T-032 · Medir latencia y coste por turno y habilitar las ablaciones   size:L  risk:MED
- **Implementa:** FR-23, FR-24 · **Decisión:** ADR-0004, supuesto A-7
- **Depende de:** T-029, T-031
- **Leer antes:** docs/evaluation-plan.md#3-brazos · docs/evaluation-plan.md#5-métricas
- **Tocar solo:** eval/src/mnemo_eval/arms.py, eval/src/mnemo_eval/cost.py, eval/src/mnemo_eval/report.py, eval/config/protocol.yaml, eval/tests/test_cost.py, test/perf/recall_perf_test.go
- **No tocar:** internal/**
- **Interfaces:**
  - *Produce:* brazos `A1` … `A9` definidos como parámetros de `recall` (lectura) o de ingesta (escritura) en `protocol.yaml`; `cost.per_turn(ingest_usage, read_usage, reader_tokens, prices) -> float`; columnas de latencia p50 / p95 / p99 y de coste en el informe; `TestRecall_P95` con etiqueta `perf`.
- **Hecho cuando:**
  - [ ] `test_cost.py` comprueba la fórmula de coste por turno de docs/evaluation-plan.md §5 con un caso calculado a mano, y que la tabla de precios lleva fecha.
  - [ ] Los percentiles se calculan sobre todas las muestras de cada brazo, no como media de percentiles (prueba).
  - [ ] Las ablaciones de lectura A1 a A6 y A9 se ejecutan sobre la ingesta del brazo M sin volver a ingerir (prueba que cuenta llamadas de escritura: cero).
  - [ ] `TestRecall_P95` carga un sujeto con 5.000 hechos y 10.000 episodios, lanza 8 clientes y falla si el p95 supera 400 ms; una segunda variante con 50.000 episodios imprime su p95 sin fallar y el dato se anota en docs/open-questions.md (cierra A-7).
  - [ ] El informe imprime, junto a cada cifra de latencia, la máquina, el tamaño del sujeto, la concurrencia y si el embebedor era local.
- **Verificar:** `uv run --project eval pytest eval/tests && docker compose up -d --wait && go test -tags=perf ./test/perf -run TestRecall_P95 -timeout 10m && uv run --project eval mnemo-eval run --protocol eval/config/smoke.yaml --arms all`
- **Fuera de alcance:** ejecutar las ablaciones con modelos reales (T-035).
- **Riesgo → mitigación:** la latencia depende de la máquina → el umbral es para la máquina de referencia descrita en el informe; en CI la prueba `perf` no se ejecuta.

### T-033 · Añadir LoCoMo como benchmark secundario   size:M  risk:LOW
- **Implementa:** FR-24 · **Decisión:** ADR-0004
- **Depende de:** T-009
- **Leer antes:** docs/evaluation-plan.md#2-conjuntos-de-datos · https://github.com/snap-research/locomo
- **Tocar solo:** eval/src/mnemo_eval/datasets/locomo.py, eval/src/mnemo_eval/cli.py, eval/config/protocol.yaml, eval/tests/test_locomo.py
- **No tocar:** eval/src/mnemo_eval/datasets/longmemeval.py
- **Hecho cuando:**
  - [ ] `fetch` descarga `locomo10.json` a `eval/data/` con comprobación de suma; el archivo no entra en git.
  - [ ] El cargador convierte cada conversación en un sujeto con sus sesiones fechadas y produce las preguntas de las categorías 1 a 4; imprime el recuento por categoría y falla si el total no es 1.540.
  - [ ] La categoría 5 se excluye del numerador y del denominador, y el informe lo dice.
  - [ ] El informe muestra dos cifras: con todas las preguntas y excluyendo las señaladas como erróneas en la lista de la auditoría, cuya ruta y suma se fijan en el protocolo.
  - [ ] El juez es el mismo modelo que para LongMemEval.
- **Verificar:** `uv run --project eval mnemo-eval fetch && uv run --project eval pytest eval/tests/test_locomo.py`
- **Fuera de alcance:** las tareas de resumen de eventos y de diálogo multimodal de LoCoMo.
- **Riesgo → mitigación:** el formato de la lista de errores de la auditoría no es utilizable → se reporta solo la cifra completa y el techo de ≈ 93,6 %, y se anota.

### T-034 · Validar el juez con 100 etiquetas propias   size:M  risk:MED
- **Implementa:** FR-24 · **Decisión:** ADR-0004, docs/evaluation-plan.md §8, ADR-0007 (decisión 14)
- **Depende de:** T-032
- **Leer antes:** docs/evaluation-plan.md#8-validación-del-juez
- **Tocar solo:** eval/src/mnemo_eval/judge_validation.py, eval/src/mnemo_eval/cli.py, eval/tests/test_judge_validation.py, eval/results/judge-validation/labels.csv, eval/results/judge-validation/summary.json
- **No tocar:** eval/src/mnemo_eval/judge.py
- **Interfaces:**
  - *Produce:* `mnemo-eval judge-sample --n 100 --out <csv>` (muestra estratificada, sin la columna del veredicto del juez); `mnemo-eval judge-validate <csv>` (calcula tasa de verdaderos positivos, de verdaderos negativos y lista los desacuerdos; falla si faltan etiquetas).
- **Hecho cuando:**
  - [ ] La muestra tiene 100 respuestas del brazo M sobre `dev60` y `write100`, mitad dadas por correctas y mitad por incorrectas por el juez, estratificadas por tipo.
  - [ ] El CSV que se etiqueta no contiene el veredicto del juez.
  - [ ] `test_judge_validation.py` comprueba las dos tasas con una matriz de confusión conocida.
  - [ ] `labels.csv` tiene las 100 etiquetas y `summary.json` las dos tasas y el número de desacuerdos.
  - [ ] Si alguna tasa es menor del 90 %, docs/evaluation-plan.md registra la advertencia que acompañará al titular.
- **Verificar:** `uv run --project eval pytest eval/tests/test_judge_validation.py && uv run --project eval mnemo-eval judge-validate eval/results/judge-validation/labels.csv`
- **Fuera de alcance:** segundo anotador (deuda D-7).
- **Riesgo → mitigación:** etiquetar viendo el veredicto sesga → la columna no existe en el archivo que se etiqueta.

### T-035 · Ejecutar la corrida completa, publicar la tabla y escribir el análisis de fallos   size:L  risk:HIGH
- **Implementa:** FR-24 · **Decisión:** ADR-0004, ADR-0007 (decisión 14)
- **Depende de:** T-033, T-034
- **Leer antes:** docs/evaluation-plan.md (completo)
- **Tocar solo:** eval/results/<protocolo>/summary.json, eval/results/<protocolo>/table.md, docs/failure-analysis.md, docs/evaluation-plan.md, README.md
- **No tocar:** internal/**, eval/src/** (el código se congela con una etiqueta antes de empezar)
- **Hecho cuando:**
  - [ ] Existe la etiqueta `eval-<protocolo>` sobre el commit con el que se ejecutó.
  - [ ] `summary.json` contiene B0, B1, B2 y M sobre el conjunto decidido en T-011, las ablaciones de lectura sobre el mismo conjunto y las de escritura sobre `write100`, con intervalos de confianza.
  - [ ] La tabla de docs/evaluation-plan.md §6 y la del README no tienen ninguna celda "pendiente" en las filas ejecutadas, e indican el identificador del protocolo, el lector, el juez y la fecha de los precios.
  - [ ] Cada hipótesis registrada aparece marcada como cumplida o no cumplida, con la cifra.
  - [ ] `docs/failure-analysis.md` tiene la taxonomía con recuentos sobre al menos 100 trazas leídas, la matriz de primera etapa fallida, tres ejemplos por categoría y la comparación con el conjunto oráculo.
  - [ ] El README enlaza las cifras de terceros como contexto y no las pone en la tabla propia.
  - [ ] El gasto total queda anotado junto al tope.
- **Verificar:** `uv run --project eval mnemo-eval verify-protocol eval/results && uv run --project eval mnemo-eval report --check-complete eval/results && ! grep -n "pending" README.md`
- **Fuera de alcance:** ajustar el sistema tras ver los resultados retenidos. Lo que se aprenda va a docs/open-questions.md para la versión siguiente.
- **Riesgo → mitigación:** tentación de retocar umbrales tras ver las 440 retenidas → el código está etiquetado antes; cualquier corrida posterior es otro protocolo con su propio identificador y se publican ambos.

### T-036 · Publicar la versión 0.1.0   size:S  risk:LOW
- **Implementa:** FR-25 · **Decisión:** docs/security-baseline.md §8, ADR-0007
- **Depende de:** T-035
- **Leer antes:** docs/quality-baseline.md#7-integración-continua · README.md
- **Tocar solo:** .github/workflows/release.yml, README.md, CONTRIBUTING.md, CHANGELOG.md, .env.example, docs/evaluation-plan.md
- **No tocar:** internal/**, eval/results/**
- **Hecho cuando:**
  - [ ] En un clon limpio, los pasos del README (copiar `.env.example`, `docker compose up -d --wait`, crear inquilino y clave, una escritura y una lectura con `curl`) funcionan tal como están escritos.
  - [ ] El README incluye un ejemplo de cómo insertar el contexto en el prompt como datos delimitados, y la advertencia de qué texto se envía a qué proveedor.
  - [ ] El flujo de versión construye la imagen, genera el SBOM en CycloneDX y adjunta `summary.json`.
  - [ ] El estado del README deja de decir "design phase".
  - [ ] La reproducción de docs/evaluation-plan.md §11 usa `docker compose up -d --wait` (F-13, ADR-0007, decisión 17).
  - [ ] `go tool govulncheck ./...` no reporta vulnerabilidades alcanzables.
- **Verificar:** `docker compose down -v && docker compose up -d --wait --build && go test -tags=e2e ./test/e2e && go tool govulncheck ./... && ! grep -n "design phase" README.md`
- **Fuera de alcance:** demo alojada, anuncio, sitio de documentación.
- **Riesgo → mitigación:** el README promete algo que no funciona en limpio → la prueba de punta a punta ejecuta los mismos pasos que el README.

---

## Definición de hecho

Para cada tarea:

**Código**
- [ ] Implementado exactamente lo de "Hecho cuando".
- [ ] Pruebas según la criticidad del componente (docs/testing-strategy.md §1); en criticidad ALTA, la prueba se vio fallar antes.
- [ ] N1, N2 y N3 pasan.
- [ ] Nada fuera de "Tocar solo" cambió (`git diff --stat` revisado).

**Verificado**
- [ ] El comando Verificar se ejecutó completo y de nuevo tras el último cambio; se leyó la salida y el código de salida.
- [ ] La salida está pegada en el commit o en el PR.

**Autorrevisión**
- [ ] Leído el diff entero, no el resumen.
- [ ] Sin SQL fuera de `internal/store`; sin contenido de usuarios en registros; sin secretos.
- [ ] Los nombres nuevos están en el glosario o se añadieron a él.

**Documentación**
- [ ] `api/openapi.yaml` actualizado si cambió una ruta.
- [ ] `docs/architecture.md` actualizado si cambió una firma pública o una tabla.
- [ ] ADR nuevo si se tomó una decisión de un solo sentido o entró una dependencia.

Para cerrar un hito: N4 pasa, la lista de riesgos del PRD se releyó, y la comprobación de consistencia de docs/plan/README.md se repitió.

## Registro de deuda

Se mantiene en [docs/quality-baseline.md](../quality-baseline.md#9-riesgos-y-deuda-técnica).

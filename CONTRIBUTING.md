# Contribuir a mnemo

Convenciones, comandos y Definición de hecho. Escrito en español, como los documentos que resume. Se lee antes de cada tarea ([docs/plan/tasks.md](docs/plan/tasks.md#cómo-se-ejecuta-este-archivo)) y se actualiza cuando cambia un comando o una convención ([docs/quality-baseline.md §8](docs/quality-baseline.md#8-documentación-obligatoria)).

Los comandos que dependen de un programa o archivo que aún no existe indican la tarea que lo crea.

## 1. Idioma

| Qué | Idioma |
|---|---|
| `docs/` y `CONTRIBUTING.md` | Español |
| `README.md` | Inglés |
| Identificadores de código y nombres de prueba | Inglés, con los términos del [glosario](docs/prd.md#glossary) tal cual: `Episode`, `Fact`, `Entity`, `Scope`, `Supersede`, `Retract`, `Purge` |
| Comentarios de documentación, mensajes de registro y textos que ve un cliente | Inglés (A-12, confirmado en [ADR-0007](docs/adr/ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md)) |
| Mensajes de commit | Español (A-11, confirmado en ADR-0007) |

## 2. Entorno

- Shell POSIX (en Windows, Git Bash), desde la raíz del repositorio.
- Go ≥ 1.26 (`go.mod` fija `toolchain go1.26.11`), Docker con Compose, `uv`.
- Secretos solo por variables de entorno; `.env` está ignorado por git. `.env.example` llega con T-036.
- El servicio `db` de Compose publica PostgreSQL en `127.0.0.1:55432` (variable `MNEMO_DB_PORT`), no en 5432, para no chocar con un PostgreSQL local. Las URL de prueba apuntan ahí: `postgres://mnemo_app:<contraseña>@127.0.0.1:55432/mnemo?sslmode=disable` y lo mismo con `mnemo_admin`; las contraseñas de desarrollo están en `docker-compose.yml` (T-002).
- Gancho de pre-commit versionado (desde T-003): se activa con `git config core.hooksPath .githooks`, una vez por clon. Ejecuta formato, `go vet`, `staticcheck` y `go test ./...`, y el commit no se crea si alguno falla. `.gitattributes` mantiene en LF los `.go` y el gancho aunque `core.autocrlf` sea `true`.

Variables:

| Variable | Para qué | Desde |
|---|---|---|
| `MNEMO_DATABASE_URL` | Conexión del servicio con el rol `mnemo_app` (obligatoria) | T-001 |
| `MNEMO_ADMIN_DATABASE_URL` | Conexión con el rol `mnemo_admin`; solo la leen `mnemo migrate`, `mnemo tenant create` y `mnemo key create`, nunca `serve` ni `mcp`. En Compose solo la tiene el servicio de una vez `admin` (ADR-0007, decisión 2) | T-002 |
| `MNEMO_LISTEN_ADDR` | Dirección de escucha; por defecto `127.0.0.1:8080` | T-001 |
| `MNEMO_LLM_BASE_URL`, `MNEMO_LLM_API_KEY`, `MNEMO_LLM_MODEL` | Modelo de chat compatible con OpenAI | T-001 |
| `MNEMO_EMBED_BASE_URL`, `MNEMO_EMBED_API_KEY`, `MNEMO_EMBED_MODEL` | Modelo de embeddings | T-001 |
| `MNEMO_EMBED_DIM` | Debe ser `1024` o el proceso no arranca (ADR-0006) | T-001 |
| `MNEMO_RERANK_URL` | Reranker opcional | T-001 |
| `MNEMO_DB_PORT` | Puerto del host en el que Compose publica la base; por defecto `55432` | T-002 |
| `MNEMO_TEST_DATABASE_URL` | Pruebas de integración, rol `mnemo_app` | T-002 |
| `MNEMO_TEST_ADMIN_DATABASE_URL` | Pruebas de integración, rol `mnemo_admin` (migraciones) | T-002 |
| `MNEMO_MCP_SUBJECT` | Sujeto de `mnemo mcp` hasta que exista la credencial | T-005, se retira en T-029 |
| `MNEMO_API_KEY` | Credencial de `mnemo mcp` por stdio | T-029 |

## 3. Módulos y límites

El mapa completo está en [docs/architecture.md §5](docs/architecture.md#5-módulos). Reglas que se rompen con facilidad:

- **SQL solo en `internal/store`**, siempre parametrizado y dentro de una transacción con `app.tenant_id` (`store.WithScope`). Ningún otro paquete importa `pgx`, salvo el raíz para el tipo del pool (AD-2).
- **Núcleos puros:** `internal/resolve`, `internal/retrieve` e `internal/policy` no importan `internal/store`, `internal/provider`, `net/http` ni `database/sql`; no leen el reloj (reciben `now`) ni generan identificadores.
- **`internal/httpapi` e `internal/mcpserver`** solo traducen hacia el paquete raíz: de los paquetes del módulo importan solo el raíz e `internal/config`. La verificación de credenciales es `(*mnemo.Memory).Authenticate` (A-14, ADR-0007). `tools/archcheck` lo aplica.
- **Operaciones sin inquilino conocido** (buscar una credencial, reclamar un trabajo, limpieza) solo por las tres funciones `SECURITY DEFINER` de ADR-0007. Añadir otra es cambiar ese ADR; `TestRLS_DefinerFunctionsApproved` falla si aparece una no aprobada.
- **El paquete raíz** no importa el SDK de MCP ni `net/http` en modo servidor.
- **Modelos solo tras los puertos** `ChatModel`, `Embedder`, `Reranker`, implementados en `internal/provider` con la biblioteca estándar. Sin SDK de proveedor ni framework de LLM (ADR-0006).
- **Solo PostgreSQL** como almacén (ADR-0001). Ninguna cola, caché ni base adicional.
- **Dependencias directas de Go: como máximo 6.** Cada una nueva lleva su justificación (nombre, licencia, mantenedores, última versión, coste de salida) en un ADR ([docs/security-baseline.md §8](docs/security-baseline.md#8-cadena-de-suministro)).

## 4. Convenciones de código

Lo que se aparta del estándar de Go ([docs/quality-baseline.md §3](docs/quality-baseline.md#3-reglas-de-código)):

- Prohibidos los paquetes `utils`, `helpers`, `common`, `shared`. `Forget` es archivar por política, nunca borrar; borrar es `Purge`.
- Errores de dominio como valores exportados: `ErrNotFound`, `ErrConflict`, `ErrInvalidState`, `ErrForbidden`. Los de infraestructura se envuelven con `%w`, contexto de operación e identificadores; nunca contenido. `internal/httpapi` los traduce al sobre de error en un único lugar.
- Toda función con E/S recibe `context.Context`; toda llamada externa lleva tiempo límite.
- Funciones de hasta 50 líneas, archivos de hasta 400, complejidad ciclomática de hasta 10.
- Comentarios que explican por qué. Las funciones exportadas del paquete raíz llevan comentario de documentación; las principales, un `Example` ejecutable.
- Sin estado global mutable; la configuración se valida entera al arrancar.
- Regla de tres antes de extraer una abstracción; ninguna interfaz con una sola implementación salvo los tres puertos de modelos.
- **Nunca se registra contenido** de episodios ni de hechos, ni el `subject_id` en claro (solo los 8 primeros caracteres de su hash), ni claves.
- **Prompts** en `internal/extract/prompts/`, con identificador de versión en la primera línea. Cambiar un prompt cambia el identificador, invalida la caché de extracción y obliga a reejecutar `dev60`.
- **Migraciones** numeradas en `internal/store/migrations/`, solo aditivas; una migración aplicada no se edita. Toda tabla con datos de inquilino lleva `tenant_id`, RLS forzada e índices que empiezan por `tenant_id` (ADR-0003).
- **Contrato primero:** `api/openapi.yaml` se cambia antes que la ruta (desde T-013).

### Nombres de prueba

- `TestFR<n>_<Comportamiento>` para cada criterio de aceptación de FR-1 a FR-23 (`TestFR6_SupersedeClosesValidity`). `tools/fr-trace` (T-031) falla si falta alguno.
- `TestProp_<Algo>` para pruebas de propiedades (`testing/quick` y fuzzing nativo).
- Una prueba cuyo nombre lleva "y" (*And*) se parte en dos.
- Etiquetas de compilación: `integration` (PostgreSQL real), `e2e` (servicio levantado con Compose), `perf` (antes de cada versión, nunca en CI), `live` (proveedores reales, nunca en CI).
- Ninguna prueba llama a un proveedor real salvo las `live`; los modelos son `FakeEmbedder` y un `ChatModel` guionizado. Ninguna usa `time.Sleep` para esperar al trabajador ni lee `time.Now()` en lógica temporal.

## 5. Flujo de trabajo

Protocolo de [docs/plan/tasks.md](docs/plan/tasks.md#cómo-se-ejecuta-este-archivo):

1. Una tarea por sesión de trabajo, un commit por tarea. Se leen este archivo, el bloque de la tarea y solo los archivos de "Leer antes".
2. Planificar primero si la tarea toca más de dos archivos o el enfoque no está claro.
3. Implementar exactamente la lista "Hecho cuando". Una idea mejor va a [docs/open-questions.md](docs/open-questions.md), no al código.
4. Ejecutar "Verificar", leer la salida completa y el código de salida, y pegar la salida en el mensaje del commit o en la descripción del PR.
5. Nunca ampliar "Tocar solo"; si hace falta otro archivo, se abre una tarea nueva.
6. Dos correcciones fallidas sobre el mismo problema: se descarta el cambio y se empieza de nuevo con el problema mejor planteado.
7. Lo aprendido que cambia el plan se anota en `docs/open-questions.md` y la tarea se replantea en el plan.
8. Hecho todo, se marca la casilla del índice de `docs/plan/tasks.md`.

En criticidad ALTA (`internal/resolve`, aislamiento, autenticación, purga) la prueba se escribe y se ve fallar antes que el código. Una corrección de error empieza por la prueba que lo reproduce.

### Commits, ramas y versiones

Convención confirmada (A-11, ADR-0007, decisión 12):

- Rama por tarea: `t-NNN-<slug>`, desde `main`, con un único commit. Se integra con `git checkout main && git merge --ff-only t-NNN-<slug> && git branch -d t-NNN-<slug> && git push origin main`.
- Primera línea: `T-NNN: <título de la tarea en el índice>`. Cuerpo: el comando Verificar y su salida, con el resultado y el código de salida. La casilla del índice se marca en el mismo commit.
- Tras el push se mira la ejecución del CI. Si falla, la corrección va en un commit `T-NNN: corrige CI (<motivo>)` antes de empezar la tarea siguiente.
- T-000 (decisiones previas, ADR-0007) sigue el mismo formato.

Criterios que exigen modelos reales, gasto o etiquetado humano (T-007, T-009, T-011, T-023, T-034, T-035): se implementa todo el código con sus pruebas con falsos; lo que falta se anota en [docs/open-questions.md §3](docs/open-questions.md#3-anotaciones-que-deja-una-tarea) con fecha y el comando que lo completa, y la tarea se marca en el índice con el sufijo "· parcial (open-questions §3)" (ADR-0007, decisión 14). Nunca se inventa una cifra.

Fijado por los documentos:

- Si una versión etiquetada empeora el p95 de `recall` más de un 20 %, o falla la suite de aislamiento o la de purga, no se integra nada más que la corrección hasta que vuelva a verde. Sin excepciones para aislamiento y purga ([docs/quality-baseline.md §5](docs/quality-baseline.md#5-objetivos-de-nivel-de-servicio)).
- Una versión es una etiqueta de git, una imagen y un binario. Las corridas del benchmark se lanzan a mano sobre un commit etiquetado `eval-<protocolo>` (T-035); nunca en CI.
- Nunca se commitean `eval/data/`, `eval/runs/` ni `eval/cache/`; sí `eval/results/<protocolo>/summary.json` y su tabla.

## 6. Comandos

Copiados de [docs/testing-strategy.md §2](docs/testing-strategy.md#2-los-cuatro-niveles-de-validación), [docs/quality-baseline.md](docs/quality-baseline.md) y [docs/plan/tasks.md](docs/plan/tasks.md). Si discrepan, manda el plan; las discrepancias conocidas están en [docs/plan/README.md](docs/plan/README.md#34-hallazgos) (F-11 y F-13; F-12 lo resuelve T-006 registrando el marcador `integration` de pytest).

### Niveles de validación

```
N1 — sintaxis y estilo (tras cada cambio de archivo)
  test -z "$(gofmt -l .)" && go vet ./... && go tool staticcheck ./...
  uv run --project eval ruff check eval && uv run --project eval ruff format --check eval

N2 — unitarias (tras cada unidad de trabajo)
  go test ./...
  uv run --project eval pytest eval/tests -m "not integration"

N3 — integración (antes de dar una tarea por hecha)
  docker compose up -d --wait db
  go test -tags=integration ./...
  go run ./tools/archcheck && go run ./tools/depbudget -max 6

N4 — dominio (antes de cerrar un hito)
  docker compose up -d --wait
  go test -tags=e2e ./test/e2e
  uv run --project eval mnemo-eval run --protocol eval/config/smoke.yaml --arms all
  go tool govulncheck ./...
  (desde M1) uv run --project eval mnemo-eval run --protocol eval/config/protocol.yaml --split dev60 --arms B2,M
```

Las líneas de Python existen desde T-006; `tools/archcheck` y `tools/depbudget`, desde T-003.

### Otras compuertas

| Qué | Comando | Desde |
|---|---|---|
| Cobertura | `go test -tags=integration -coverprofile=coverage.out ./... && go run ./tools/covergate -profile coverage.out` | T-031 |
| Trazabilidad de requisitos | `go run ./tools/fr-trace -from 1 -to 23` | T-031 |
| Gancho de pre-commit | `.githooks/pre-commit` (se activa con `git config core.hooksPath .githooks`) | T-003; líneas de Python, T-006 |
| Python completo | `uv run --project eval ruff check eval && uv run --project eval ruff format --check eval && uv run --project eval pytest eval/tests` | T-006 |
| Vulnerabilidades | `go tool govulncheck ./... && uv run --project eval pip-audit` | T-001 / T-006 |
| Arranque con un comando | `docker compose up -d --wait && curl -fsS http://127.0.0.1:8080/readyz` | T-003 |
| Aislamiento (incluye las funciones `SECURITY DEFINER` aprobadas) | `go test -tags=integration ./internal/store -run 'TestIsolation_\|TestRLS_\|TestAppRole_' -count=1` | T-012 |
| Purga | `go test ./internal/store -tags=integration -run TestPurge_NoRowsLeftAnyTable` | T-028 |
| Latencia de `recall` | `go test -tags=perf ./test/perf -run TestRecall_P95 -timeout 10m` | T-032 |

En la tabla, `\|` es la barra vertical escapada para Markdown; en la shell se escribe `|`. Las demás funciones de aptitud, con su comando, están en [docs/quality-baseline.md §1](docs/quality-baseline.md#1-funciones-de-aptitud).

### Servicio y harness

```
mnemo migrate
mnemo serve
mnemo mcp
mnemo tenant create <nombre>
mnemo key create --tenant <id> --scopes <lista> [--subject <id>]
```

`migrate`, `tenant create` y `key create` necesitan `MNEMO_ADMIN_DATABASE_URL`. Con Compose se ejecutan en un contenedor efímero del servicio `admin`, no en el servicio `mnemo`: `docker compose run --rm -T admin tenant create <nombre>`.

Reproducción del benchmark ([docs/evaluation-plan.md §11](docs/evaluation-plan.md#11-reproducción)):

```
docker compose up -d --wait
uv run --project eval mnemo-eval fetch
uv run --project eval mnemo-eval run --protocol eval/config/protocol.yaml --arms all
uv run --project eval mnemo-eval report --out eval/results/
```

Comprobación de que los resultados no mezclan protocolos: `uv run --project eval mnemo-eval verify-protocol eval/results`.

## 7. Definición de hecho

Para cada tarea ([docs/plan/tasks.md](docs/plan/tasks.md#definición-de-hecho)):

**Código**
- [ ] Implementado exactamente lo de "Hecho cuando".
- [ ] Pruebas según la criticidad del componente ([docs/testing-strategy.md §1](docs/testing-strategy.md#1-tipos-de-prueba-por-componente)); en criticidad ALTA, la prueba se vio fallar antes.
- [ ] N1, N2 y N3 pasan.
- [ ] Nada fuera de "Tocar solo" cambió (`git diff --stat` revisado).

**Verificado**
- [ ] El comando Verificar se ejecutó completo y de nuevo tras el último cambio; se leyó la salida y el código de salida.
- [ ] La salida está pegada en el commit o en el PR.

**Autorrevisión** (no hay segundo revisor, supuesto A-8)
- [ ] Leído el diff entero, no el resumen.
- [ ] Sin SQL fuera de `internal/store`; sin contenido de usuarios en registros; sin secretos.
- [ ] Los nombres nuevos están en el glosario o se añadieron a él.
- [ ] Cada prueba nueva falló antes de escribir el código, y deja de pasar si se borra su aserción.
- [ ] Ninguna prueba es solo una comparación con instantánea, ni hay puntos de entrada solo para pruebas en código de producción.

**Documentación**
- [ ] `api/openapi.yaml` actualizado si cambió una ruta.
- [ ] `docs/architecture.md` actualizado si cambió una firma pública o una tabla.
- [ ] ADR nuevo si se tomó una decisión de un solo sentido o entró una dependencia.
- [ ] `docs/evaluation-plan.md` con su registro de cambios si cambió el protocolo; `README.md` si cambió el arranque, la API o la tabla de resultados; este archivo si cambió un comando o una convención.

Para cerrar un hito: N4 pasa, la lista de riesgos del PRD se releyó y la comprobación de consistencia de [docs/plan/README.md](docs/plan/README.md#3-comprobación-de-consistencia) se repitió.

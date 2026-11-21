# ADR-0007: Operaciones sin inquilino conocido y ajustes del plan antes de T-001

## Estado
Aceptado — 2026-11-21. Completa ADR-0003 y sustituye su nota de implementación sobre la comparación de hashes en tiempo constante (decisión 2); el resto de ADR-0003 sigue vigente.

## Reversibilidad
- **Tipo de puerta:** mixta. Las funciones `SECURITY DEFINER` (decisión 2) y la forma de `fact_sources` (decisión 4) son de un solo sentido en la práctica: cambiarlas con datos dentro es una migración y una auditoría de permisos. El resto (reglas de importación, listas "Tocar solo", orden de tareas, convenciones) es de dos sentidos.
- **Coste de revertir:** funciones de definidor, una migración que las retire y otra vía para las tres operaciones (unos días). `fact_sources`, una migración con copia de datos.
- **Disparador de revisión:** hace falta una cuarta operación sin inquilino conocido; o una auditoría encuentra una forma de usar una de las funciones para leer o escribir filas de un inquilino distinto del que devuelve.

## Contexto
La comprobación de consistencia del 2026-11-20 ([plan/README.md §3.4](../plan/README.md#34-hallazgos)) dejó tres hallazgos HIGH y varios MEDIUM y LOW con preguntas abiertas Q-3 y Q-13 a Q-24 ([open-questions.md](../open-questions.md)). El autor delegó las decisiones el 2026-11-21. Este ADR las registra todas antes de T-001 para que ninguna tarea tenga que decidir sobre la marcha ni ampliar su propia lista "Tocar solo" (regla 5 del plan).

El problema de fondo (Q-13, F-1): con RLS forzada y un rol de aplicación sin `BYPASSRLS` (ADR-0003), una consulta sin `app.tenant_id` falla cerrada. Tres operaciones no conocen el inquilino de antemano: buscar una credencial por su hash (T-012), tomar el siguiente trabajo con turno entre inquilinos (T-015) y la limpieza por antigüedad (T-016).

Las demás (Q-14 a Q-24) son huecos de redacción del plan: reglas de importación contradictorias, una clave primaria imposible, listas "Tocar solo" incompletas, una dependencia que falta y etapas de CI sin dueño.

## Impulsores de la decisión
* **Imprescindible** — el rol `mnemo_app` sigue sin `BYPASSRLS` y sin propiedad de tablas; el fallo cerrado de ADR-0003 no se debilita para el camino normal.
* **Imprescindible** — la superficie que salta RLS es finita, nombrada y comprobada por una prueba que falla si crece.
* **Imprescindible** — borrar un episodio nunca deja un hecho sin procedencia legible (FR-11) ni rompe una restricción.
* **Deseable** — ninguna dependencia nueva y ningún proceso ni pool extra.

## Decisión

### 1. Ruta del módulo (Q-3, A-1)
El módulo es `github.com/DrewGGM/mnemo`. Confirmado.

### 2. Operaciones sin inquilino conocido (Q-13, A-13)
Tres funciones `SECURITY DEFINER` en el esquema `public`, propiedad de `mnemo_admin`, son la única vía por la que `mnemo_app` toca filas sin `SET LOCAL app.tenant_id`:

| Función | Qué hace | Devuelve | La crea |
|---|---|---|---|
| `mnemo_lookup_api_key(p_key_hash bytea)` | Busca la credencial no revocada con ese hash | `tenant_id`, `key_id`, `scopes`, `subject_id`; nunca el hash | T-012, en `0003_tenancy.sql` |
| `mnemo_claim_job()` | Pasa a `failed` los trabajos abandonados; toma el siguiente trabajo listo con turno entre inquilinos y `FOR UPDATE SKIP LOCKED`, lo pasa a `running`, fija `locked_at` y suma un intento | La fila del trabajo, incluido su `tenant_id` | T-015, en `0005_jobs.sql` |
| `mnemo_cleanup()` | Borra `jobs` terminados de más de 30 días salvo los `extract` en `failed` (decisión 4), `idempotency_keys` de más de 24 h y `usage_events` de más de 180 días | Un recuento por tabla | T-016, en `0006_usage.sql` |

Reglas comunes:

- Cada función se declara con `SET search_path = pg_catalog, public, pg_temp`, nombra las tablas con su esquema y no ejecuta SQL dinámico.
- `REVOKE ALL ON FUNCTION … FROM PUBLIC` y `GRANT EXECUTE … TO mnemo_app`. Aparte del dueño, ningún otro rol tiene `EXECUTE`.
- No reciben inquilino, sujeto, umbrales ni hora como parámetro. Los umbrales (5 min para retomar un trabajo bloqueado y 5 intentos como máximo; 30 días, 24 h y 180 días para la limpieza) van fijos en el cuerpo, y la hora es la de la base (`now()`). Así nadie que llame puede ampliar lo que borran ni lo que devuelven. Las pruebas que necesitan filas antiguas las fechan con la conexión de `mnemo_admin`.
- **Trabajo listo:** `queued` con `run_after ≤ now()`, o `running` con `locked_at` de hace más de 5 min; en ambos casos con `attempts < 5`.
- **Trabajos abandonados:** antes de reclamar, `mnemo_claim_job()` pasa a `failed`, con `error = 'abandoned'` y `finished_at = now()`, los `running` con `locked_at` de hace más de 5 min y `attempts ≥ 5`. Un trabajo que tumba o cuelga al trabajador no se retoma sin fin.
- **Turno entre inquilinos:** el último reclamo de un inquilino es `max(locked_at)` de sus trabajos. `locked_at` se fija al reclamar y no se borra nunca, tampoco al pasar a `done`, `failed` o de vuelta a `queued`. Se elige el inquilino con trabajos listos cuyo último reclamo es más antiguo, con los nulos primero y empate por `tenant_id`; dentro de él, el trabajo listo más antiguo por `(run_after, created_at, id)`. PostgreSQL no admite `FOR UPDATE` en el mismo nivel que un agregado, así que la agregación va en una subconsulta: `UPDATE public.jobs … WHERE id = (SELECT j.id FROM public.jobs j JOIN (SELECT tenant_id, max(locked_at) AS last_claim FROM public.jobs GROUP BY tenant_id) t USING (tenant_id) WHERE <listo> ORDER BY t.last_claim NULLS FIRST, j.tenant_id, j.run_after, j.created_at, j.id LIMIT 1 FOR UPDATE OF j SKIP LOCKED) RETURNING …`.
- **Un solo reloj en la cola:** el reintento de T-015 fija `run_after` en SQL (`now() + make_interval(secs => $1)`); de Go solo llega la duración de la espera. Las pruebas mueven el tiempo fechando filas con `mnemo_admin`, nunca con un reloj de Go.
- **Un solo dueño del tope de intentos.** Tras un fallo, SQL decide en una sola sentencia si queda intento: `SET status = CASE WHEN attempts >= 5 THEN 'failed' ELSE 'queued' END, run_after = now() + make_interval(secs => $espera), error = $error`. Go no compara `attempts`, así que ningún trabajo con `attempts = 5` vuelve a `queued`, donde no estaría listo ni lo retiraría nadie.
- **Arriendo con dueño.** El manejador corre con un contexto de plazo de 4 min, menor que el arriendo de 5 min. Todo cambio de estado posterior al reclamo (`done`, reintento o `failed`) lleva la condición `WHERE id = $id AND status = 'running' AND attempts = $intento_reclamado`. Si afecta a 0 filas, otro ciclo retomó el trabajo: el trabajador lo descarta sin error y sin reintento, y no pisa el estado de la segunda ejecución.
- Lo que sigue a cada función va bajo `store.WithScope` con el inquilino devuelto: el manejador del trabajo, el cambio de estado a `done` o `failed` y cualquier lectura posterior a la autenticación.
- `FORCE ROW LEVEL SECURITY` también se aplica al dueño de la tabla. Por eso `mnemo_admin` se crea con `BYPASSRLS` y `CREATEDB` en `deploy/initdb/01-roles.sql` (T-002). Es el rol de migraciones y de los subcomandos de administración, nunca el del servicio. `mnemo_app` no tiene `CREATE` sobre el esquema `public`.
- **Credencial de `mnemo_admin` fuera del servicio.** Su URL va en `MNEMO_ADMIN_DATABASE_URL`, que solo leen `mnemo migrate`, `mnemo tenant create` y `mnemo key create` desde `cmd/mnemo`; `config.Load` no la lee, y `mnemo serve` y `mnemo mcp` ni la leen ni la necesitan. En Compose, las migraciones corren en el servicio de una vez `admin` (imagen de `mnemo`, orden por defecto `migrate`), el único que recibe esa variable; `mnemo` depende de él con `condition: service_completed_successfully` y solo tiene `MNEMO_DATABASE_URL`, con el rol `mnemo_app`. Los subcomandos de administración se ejecutan en un contenedor efímero de ese servicio: `docker compose run --rm -T admin tenant create <nombre>` (T-002 y T-003).
- **Privilegios de tabla mínimos.** `mnemo_app` recibe, tabla por tabla, `SELECT, INSERT, UPDATE, DELETE` o menos (solo `INSERT` en `audit_log`; `SELECT` e `INSERT` en `instance_settings`), nunca `GRANT ALL` ni `ALTER DEFAULT PRIVILEGES … GRANT ALL`. `TRUNCATE` no está sujeto a RLS: con él, `mnemo_app` vaciaría las tablas de todos los inquilinos. `TRIGGER` le permitiría colgar código de las tablas.
- Una vista se evalúa con los permisos de su dueño. Como `mnemo_admin` es dueño de todo y tiene `BYPASSRLS`, una vista suya que pudiera leer `mnemo_app` saltaría RLS para todos los inquilinos. Por eso `mnemo_app` no recibe ningún privilegio sobre vistas materializadas ni sobre vistas sin `security_invoker = true`.
- **Funciones de aptitud**, en `internal/store/isolation_test.go` y dentro del comando de aislamiento (filtros `TestRLS_` y `TestAppRole_`), las tres de T-012:
  - `TestRLS_DefinerFunctionsApproved` recorre las funciones con `prosecdef` de todos los esquemas que no son del sistema (todos salvo `pg_catalog`, `information_schema`, `pg_toast` y los temporales). Exige que coincidan, con sus firmas, con la lista aprobada; que su dueño sea `mnemo_admin`; que `proconfig` sea exactamente `{"search_path=pg_catalog, public, pg_temp"}`; y que los roles con `EXECUTE` según `aclexplode(proacl)` sean exactamente `mnemo_admin` y `mnemo_app`, sin `PUBLIC` ni ningún otro. T-015 y T-016 añaden su función a la lista en la misma prueba.
  - `TestRLS_NoOwnerPrivilegedViews` falla si `mnemo_app` tiene algún privilegio sobre una vista materializada o sobre una vista sin `security_invoker = true`, en cualquier esquema que no sea del sistema.
  - `TestAppRole_NotOwnerNoBypass` falla si `mnemo_app` es dueño de algo, tiene `BYPASSRLS` o `CREATE` sobre `public`, o si `has_table_privilege('mnemo_app', c.oid, 'TRUNCATE')` o `has_table_privilege('mnemo_app', c.oid, 'TRIGGER')` es verdadero para alguna tabla de un esquema que no sea del sistema.
- **Pruebas de la cola y de la limpieza en una base propia.** `mnemo_claim_job()` y `mnemo_cleanup()` actúan sobre todos los inquilinos de la base. Una prueba que las llama, directamente, por `store.ClaimJob` o `store.Cleanup`, o al ejecutar el trabajador, no comparte base con otras pruebas ni con el servicio de Compose, cuyo trabajador también reclama: si no, reclamaría o borraría filas ajenas, y los trabajos `queued` que otras pruebas dejan sin reclamar romperían el turno de `TestJobs_ClaimTakesTurns`. `storetest.IsolatedDB(t)`, en `internal/store/storetest/storetest.go` (T-015), crea con `mnemo_admin` la base `mnemo_t_<sufijo aleatorio>`, le aplica `store.Migrate`, devuelve las URL de `mnemo_app` y de `mnemo_admin` hacia ella y la borra en `t.Cleanup` con `DROP DATABASE … WITH (FORCE)`. Ningún trabajador de Compose apunta a ella. Las demás pruebas de integración siguen en la base de desarrollo, cada una con su inquilino (AD-5).
- **Credencial y tiempo constante.** La búsqueda es una igualdad sobre el SHA-256 del secreto presentado. ADR-0003 y TH-6 piden comparación en tiempo constante: aquí el tiempo de la búsqueda depende del hash y no del secreto. Aunque una medición revelara bytes del hash guardado, conocer el hash no sirve para autenticarse: haría falta una preimagen de SHA-256 de un secreto de 256 bits. Se acepta la igualdad en la base como equivalente.

### 3. Reglas de importación de HTTP y MCP (Q-14, A-14)
`internal/httpapi` e `internal/mcpserver` importan solo el paquete raíz e `internal/config`. Es la regla que aplica `tools/archcheck` (T-003). La verificación de credenciales se expone en el paquete raíz como `(*mnemo.Memory).Authenticate(ctx, plaintext string) (mnemo.Principal, error)`, que envuelve `store.LookupKey`. Esa es la firma, ya escrita en [architecture.md §8](../architecture.md#8-api-pública-del-paquete-go); si T-012 necesitara otra, se anota en open-questions.md y se replantea la tarea (regla 7 del plan). `cmd/mnemo` sí puede importar `internal/store`, porque es la composición. `tools/archcheck` mira también las importaciones de los archivos de prueba, y en ellas admite además `internal/store/storetest` (decisión 2). [architecture.md §5](../architecture.md#5-módulos) queda corregido.

### 4. Procedencia que sobrevive al borrado de episodios (Q-15)
```sql
fact_sources(id bigserial pk, tenant_id,
             fact_id uuid not null references facts on delete cascade,
             episode_id uuid null references episodes on delete set null,
             session_id text not null, seq int not null,          -- instantánea de procedencia
             occurred_at timestamptz not null, role text not null,
             unique (tenant_id, fact_id, session_id, seq))
```

- Clave primaria sustituta. `episode_id` admite nulos y pasa a nulo si el episodio se borra. La unicidad `(fact_id, session_id, seq)` se declara con `tenant_id` delante, como todo índice (ADR-0003), y significa lo mismo porque `fact_id` ya determina el inquilino.
- `session_id`, `seq`, `occurred_at` y `role` se copian del episodio al insertar (T-010) y no se actualizan nunca. No contienen texto del episodio.
- Al resumir (T-026), las filas de `fact_sources` de los episodios que se van a borrar se reapuntan al episodio `summary` en la misma transacción, antes de borrar los originales. La instantánea conserva la ronda original.
- Una sesión se puede resumir (T-026) si ningún trabajo `extract` de ese sujeto y esa sesión está `queued`, `running` o `failed`; la consulta filtra por `tenant_id`, `subject_id` y `payload->>'session_id'`. Un trabajo `done` que la limpieza ya borró cuenta como terminado. Por eso `mnemo_cleanup()` no borra los `extract` en `failed`: sin ellos, una sesión cuya extracción falló parecería extraída. Los borra la purga del sujeto.
- La deduplicación de `jobs` incluye al sujeto: `subject_id text not null` y `unique (tenant_id, subject_id, kind, dedupe_key)` (T-015, `0005_jobs.sql`), y el trabajo `extract` lleva `session_id` en `payload`. `session_id` solo es único dentro de un sujeto, así que con la unicidad anterior, `(tenant_id, kind, dedupe_key)`, el lote de un segundo sujeto con el mismo `session_id` y las mismas rondas se deduplicaba contra el del primero: sus episodios no se extraían nunca y, sin trabajo propio, su sesión parecería extraída y se resumiría. Lo mismo valía para `consolidate` y `forget`, cuya clave es la fecha UTC.
- Al fusionar duplicados (T-026), el hecho conservado recibe la unión de las procedencias con `on conflict do nothing` sobre la unicidad.
- FR-11 se cumple con la instantánea: todo hecho devuelto trae al menos una procedencia con `session_id`, `seq`, `occurred_at` y `role`, y además `episode_id` si el episodio sigue existiendo.
- La purga borra `fact_sources` en cascada desde `facts`.

### 5. Claves foráneas hacia `entities` (Q-16, A-15)
`0002_facts.sql` (T-010) crea `subject_entity_id` y `object_entity_id` sin clave foránea; `0007_entities.sql` (T-017) añade las restricciones con `ALTER TABLE`. Confirmado.

### 6. Listas "Tocar solo" (Q-17, F-5)
Se corrigen en [plan/tasks.md](../plan/tasks.md) en este mismo cambio, antes de empezar ninguna tarea:

| Tarea | Se añade a "Tocar solo" |
|---|---|
| T-001 | `go.sum` (herramientas de la directiva `tool`) |
| T-002 | `go.mod`, `go.sum` (pgx) |
| T-003 | `.githooks/pre-commit`, `CONTRIBUTING.md` (decisión 9) |
| T-004 | `go.mod`, `go.sum` (pgvector-go) |
| T-005 | `go.sum` |
| T-006 | `.github/workflows/ci.yml`, `.githooks/pre-commit` (decisión 9) |
| T-010 | `internal/store/facts_test.go` (`TestFacts_SourceSurvivesEpisodeDelete`, decisión 4) |
| T-012 | `mnemo.go` (decisión 3), `eval/src/mnemo_eval/client.py`, `test/e2e/skeleton_test.go`, `.github/workflows/ci.yml`, `CONTRIBUTING.md` (credencial del harness: decisión 16) |
| T-015 | `internal/store/isolation_test.go`, `internal/store/storetest/storetest.go` (base propia de las pruebas de la cola; decisión 2) |
| T-016 | `internal/jobs/cleanup.go`, `internal/jobs/cleanup_test.go`, `internal/store/isolation_test.go`, `internal/store/settings.go`, `internal/store/settings_test.go`, `cmd/mnemo/main.go` (decisiones 2 y 11) |
| T-020 | `options.go` |
| T-024 | `internal/httpapi/recall_test.go` (`TestFR11_ProvenanceOnEveryItem`, decisión 4) |
| T-025 | `mnemo.go`, `options.go`, `eval/src/mnemo_eval/cli.py` |
| T-026 | `mnemo.go`, `options.go`, `internal/store/consolidate.go` (decisión 4) |
| T-027 | `mnemo.go`, `options.go` |
| T-028 | `internal/httpapi/subjects_test.go` (`TestFR16_` por HTTP: segunda purga, `recall` posterior y 401 de la credencial purgada; decisión 10) |
| T-030 | `docs/architecture.md` (su "Hecho cuando" actualiza §8 en el mismo commit si cambia un identificador) |
| T-031 | `tools/depbudget/main.go`, `tools/depbudget/main_test.go` (decisión 11) |
| T-036 | `docs/evaluation-plan.md` (`--wait` en §11, F-13; decisión 17) |

T-029 no cambia de lista: consume `(*mnemo.Memory).Authenticate` en lugar de `store.LookupKey` (decisión 3).

La limpieza de T-016 no es una fila de `jobs`, porque no tiene inquilino: es una tarea periódica del trabajador (`internal/jobs/cleanup.go`) que llama a `store.Cleanup` y, a través de ella, a `mnemo_cleanup()`.

### 7. `release` y `audit_log` (Q-18, F-6)
T-025 depende de T-028. Se ejecuta T-028 antes que T-025; la numeración no cambia.

### 8. Trazabilidad de FR-9 (Q-20, F-8)
La prueba de fuentes desactivables de T-021 se llama `TestFR9_SourcesToggle`, en lugar de `TestRecall_SourcesToggle`.

### 9. Gancho de pre-commit y etapas de CI sin dueño (Q-19, F-7, F-12)
- **T-003** crea `.githooks/pre-commit`, versionado y ejecutable, con las líneas de Go de [quality-baseline.md §2](../quality-baseline.md#2-sensores) marcadas como de pre-commit: formato, `go vet`, `staticcheck` y `go test ./...`. También documenta en `CONTRIBUTING.md` cómo activarlo: `git config core.hooksPath .githooks`. Además, el servicio PostgreSQL del CI crea los roles con `deploy/initdb/01-roles.sql`, igual que Compose, para que las pruebas de integración del CI vean `mnemo_admin` y `mnemo_app` con los mismos atributos.
- **T-006** añade al gancho las líneas de Python (`ruff check`, `ruff format --check`, `pytest -m "not integration"`) y al CI las etapas de Python: `uv sync --frozen`, `ruff`, `pytest` y `pip-audit`. Declara `pip-audit` como dependencia de desarrollo en `eval/pyproject.toml`, registra el marcador `integration` de pytest en `eval/pyproject.toml` y añade al CI la prueba de punta a punta (`go test -tags=e2e ./test/e2e`) y la evaluación mínima (`mnemo-eval run --protocol eval/config/smoke.yaml --arms all`).

### 10. Supresión de credenciales y claves de idempotencia (Q-22, F-9)
- `idempotency_keys` lleva `subject_id text not null` desde su migración (`0004_idempotency.sql`, T-014). La purga la descubre por esa columna, como a las demás. Su clave primaria es `(tenant_id, subject_id, key)`: la misma `Idempotency-Key` usada por dos sujetos del mismo inquilino no choca, y un sujeto no puede averiguar ni ocupar de antemano las claves de otro (TH-2).
- `api_keys.subject_id` también se descubre: la purga borra las credenciales ligadas al sujeto purgado, y el recibo cuenta `api_keys` e `idempotency_keys`. Una purga hecha con una credencial ligada a ese sujeto deja de autenticar en cuanto se confirma.
- Por eso la idempotencia de la purga (FR-16) se comprueba con una credencial del inquilino no ligada a sujeto: la segunda purga devuelve el recibo con ceros y el `recall` posterior, vacío con `no_evidence`. Con la credencial ligada al sujeto purgado, ambas peticiones reciben 401 `UNAUTHENTICATED` (T-028).
- Queda escrito en [security-baseline.md §6](../security-baseline.md#6-datos-personales-y-supresión).

### 11. Funciones de aptitud sin tarea (Q-21, F-10)
- **Modelo de embeddings registrado (ADR-0006, decisión 4) → T-016.** `0006_usage.sql` crea `instance_settings(key text primary key, value text not null)`. Es una tabla sin `tenant_id` y sin datos de inquilino, y `mnemo_app` solo tiene `SELECT` e `INSERT` sobre ella. `store.EnsureSetting(ctx, pool, key, value string) error` hace `INSERT … ON CONFLICT (key) DO NOTHING` y después `SELECT`: registra el valor si no hay ninguno, no choca si dos procesos arrancan a la vez y falla, nombrando ambos valores, si el registrado es otro. `store.EnsureEmbedModel(ctx, pool, model)` la llama con la clave `embed_model`. Al arrancar, `mnemo serve` y `mnemo mcp` llaman a `store.EnsureEmbedModel` y terminan con código distinto de cero si falla. La comprobación vive en `cmd/mnemo` y no en `mnemo.Open`, para que las pruebas de integración con `FakeEmbedder` compartan la base de desarrollo. Prueba: `TestSettings_EmbedModelMismatchFails`, que llama a `store.EnsureSetting` con una clave propia de la ejecución (sufijo aleatorio) y la borra al terminar con `mnemo_admin`; nunca toca `embed_model`, que el servicio de desarrollo ya registró.
- **Lista negra de clientes de otras bases (ADR-0001) → T-031.** `tools/depbudget` falla si `go.mod` requiere un cliente de otra base de datos, cola o caché. Como mínimo: `github.com/go-sql-driver/mysql`, `github.com/lib/pq`, `github.com/mattn/go-sqlite3`, `modernc.org/sqlite`, `go.mongodb.org/mongo-driver`, `github.com/redis/go-redis`, `github.com/gomodule/redigo`, `github.com/nats-io/nats.go`, `github.com/qdrant/go-client`, `github.com/weaviate/weaviate-go-client`, `github.com/neo4j/neo4j-go-driver`. Incluye un caso negativo en `tools/depbudget/main_test.go`.

### 12. Idioma y convención de commits (Q-23, A-12; Q-24, A-11)
- A-12 confirmado: identificadores, comentarios de documentación, registros y textos que ve un cliente, en inglés. Los ejemplos en español de `docs/` son ilustraciones.
- A-11 confirmado: una rama `t-NNN-<slug>` desde `main` por tarea, un único commit con primera línea `T-NNN: <título del índice>` y, en el cuerpo, el comando Verificar con su salida. Se integra con `git merge --ff-only` en `main` y `git push origin main`. Si el CI de ese push falla, la corrección va en un commit `T-NNN: corrige CI (<motivo>)` antes de empezar la tarea siguiente. T-000, la tarea que registra este ADR, usa el mismo formato.

### 13. Dependencia de M2 respecto a M1 (F-16)
T-012 depende también de T-011, pero solo de la decisión registrada y no de su código. Las corridas reales de T-011 quedan pendientes del autor, porque no hay claves de proveedor (decisión 14). Por eso **M2 empieza con la decisión de M1 aplazada**. El riesgo es R-1 del [PRD](../prd.md#dependencies--risks): si la extracción no supera a B2, el trabajo de M2 y M3 sobre hechos (T-017 a T-020, T-025 a T-027) se habrá construido sobre una hipótesis no medida, y con el plan B parte de ese trabajo sobraría. Se acepta porque M2 (inquilinos, autenticación, idempotencia, cola, medición) sirve igual con el plan B. La primera corrida real de T-011 se hace antes de la primera de T-035 y su decisión se registra en Q-1.

### 14. Criterios que exigen modelos reales, dinero o etiquetado humano
Afecta a T-007 (prueba `live`), T-009 (corrida real sobre `dev60`), T-011, T-023 (prueba `live`), T-034 (100 etiquetas) y T-035. El código y sus pruebas con falsos se implementan completos. Cada criterio que no se puede cumplir sin claves, gasto o etiquetado humano queda en [open-questions.md §3](../open-questions.md#3-anotaciones-que-deja-una-tarea), con la fecha, lo que falta y el comando exacto que lo completa. La tarea se marca en el índice con la casilla hecha y el sufijo "· parcial (open-questions §3)". Ninguna cifra se inventa ni se estima en su lugar.

### 15. Puerto de la base de desarrollo y pruebas
El servicio `db` de Compose publica en `127.0.0.1:${MNEMO_DB_PORT:-55432}`, no en 5432, para no chocar con un PostgreSQL local. `MNEMO_TEST_DATABASE_URL` y `MNEMO_TEST_ADMIN_DATABASE_URL` apuntan a ese puerto. Lo fija T-002.

### 16. Credencial del harness desde T-012
Desde T-012 toda ruta `/v1` exige `Bearer`. `MnemoClient` toma la credencial de `MNEMO_EVAL_API_KEY` cuando no recibe `api_key`, así que `cli.py` y `pipeline.py` no cambian. En el CI, antes de la evaluación mínima, se crean un inquilino y una clave con `docker compose run --rm -T admin tenant create` y `docker compose run --rm -T admin key create`, en un contenedor efímero que lleva `MNEMO_ADMIN_DATABASE_URL` (decisión 2), y la clave se pasa en esa variable. El servicio `mnemo` en marcha no tiene esa credencial. `CONTRIBUTING.md` explica lo mismo para N4. Todo lo hace T-012, que por eso lista `.github/workflows/ci.yml` y `CONTRIBUTING.md` (decisión 6).

### 17. Hallazgos LOW que se cierran ahora (F-11, F-13, F-14, F-15)
Dejarlos para una tarea posterior exigía que esa tarea tocara un documento fuera de su lista "Tocar solo". Los que caen en documentos de T-000 se corrigen en este cambio:

- **F-11:** [quality-baseline.md §2](../quality-baseline.md#2-sensores) usa `go run ./tools/covergate -profile coverage.out`, como T-031 y `CONTRIBUTING.md`.
- **F-14:** [architecture.md §5](../architecture.md#5-módulos) usa las firmas del plan: `extract.Extract(ctx, chat, w)` (T-010) y `resolve.Plan(c, existing, v, now) ([]Transition, error)` (T-018). En §8, `mnemo.Migrate(ctx, adminURL string) error` envuelve `store.Migrate` y recibe la URL de `mnemo_admin`, no el pool de la aplicación, porque migrar exige el rol dueño. La crea T-030, que lista `mnemo.go`.
- **F-15:** el criterio de T-002 dice "las de datos con `tenant_id`" y deja fuera a `tenants`.
- **F-13:** `evaluation-plan.md` queda fuera de T-000. T-036 lo añade a su lista y tiene el criterio de poner `--wait` en §11.

## Consecuencias
### Positivas
- El fallo cerrado de ADR-0003 sigue intacto para todo el camino normal; lo que lo salta son tres funciones nombradas con una prueba que vigila la lista.
- Ningún proceso, pool ni rol de servicio adicional.
- Borrar episodios ya no rompe la procedencia, y FR-14 deja de estar bloqueado.
- Todas las tareas tienen sus listas "Tocar solo" completas antes de empezar; la regla 5 del plan se puede cumplir al pie de la letra.

### Negativas
- `mnemo_admin` pasa a tener `BYPASSRLS`. Una credencial de migraciones filtrada lee todo; ya era dueña de las tablas y podía desactivar RLS, así que el riesgo no cambia de clase, pero queda escrito. Además, todo lo que crean las migraciones se evalúa con un dueño que salta RLS, y eso obliga a vigilar las vistas (decisión 2).
- Tres piezas de lógica viven en PL/pgSQL o SQL de funciones, fuera de los archivos Go de `internal/store`. Siguen en `internal/store/migrations/`, así que la regla "SQL solo en `internal/store`" se mantiene.
- `fact_sources` duplica cuatro columnas del episodio.
- M2 empieza sin la evidencia de M1 (decisión 13).
- Cada prueba que reclama trabajos o limpia crea, migra y borra su propia base: esas pruebas tardan más.
- Compose gana el servicio de una vez `admin`, que es lo que mantiene la credencial de `mnemo_admin` fuera del proceso del servicio.

### Riesgos
- Una función de definidor con `search_path` mal fijado permite secuestrar un operador o una función → `SET search_path` explícito, nombres con esquema, `mnemo_app` sin `CREATE` en `public` y `TestRLS_DefinerFunctionsApproved`.
- Alguien añade un parámetro de inquilino "por comodidad" a una función → la prueba compara también las firmas con la lista aprobada.
- Una migración futura crea una vista o una vista materializada y se la concede a `mnemo_app`: como su dueño tiene `BYPASSRLS`, leería filas de todos los inquilinos (TH-1) sin que `TestIsolation_AllTables` lo vea, porque esa prueba solo recorre tablas → `TestRLS_NoOwnerPrivilegedViews`; toda vista que lea `mnemo_app` se declara `security_invoker = true`.
- Una migración concede `GRANT ALL` a `mnemo_app` por comodidad y, con `TRUNCATE`, vacía las tablas de todos los inquilinos sin pasar por RLS → privilegios tabla por tabla y `TestAppRole_NotOwnerNoBypass`, que falla con `TRUNCATE` o `TRIGGER`.
- La función de reclamar trabajos se vuelve lenta con muchos inquilinos → índice `(tenant_id, status, run_after)` en `jobs`, que respeta la regla de ADR-0003 de empezar por `tenant_id`; con un solo trabajador y pocos inquilinos no es un problema en v1.

### Puntos de sensibilidad
- La lista de funciones aprobadas es parte del modelo de aislamiento: cambiarla es cambiar este ADR.
- La instantánea de `fact_sources` no se actualiza nunca; si un episodio pudiera editarse dejaría de ser fiel, pero los episodios son inmutables (ADR-0002).

### Puntos de compromiso
- **Aislamiento contra simplicidad operativa:** funciones de definidor de superficie mínima frente a un segundo pool con un rol privilegiado en el trabajador.
- **Velocidad contra evidencia:** empezar M2 sin la decisión de M1.

## Alternativas consideradas
1. **Segundo pool con el rol de migraciones para el trabajador y la autenticación** — rechazada: el proceso del servicio tendría siempre a mano una conexión que salta RLS, y cualquier consulta por ese pool sería una fuga posible. Con las funciones, solo tres consultas fijas la saltan.
2. **Política RLS con una excepción para `app.tenant_id` vacío** — rechazada: convierte el olvido de `SET LOCAL` en lectura de todo, que es justo el fallo que ADR-0003 quiere hacer imposible.
3. **Buscar la credencial por prefijo y comparar el hash en Go** — rechazada: el prefijo no es secreto y la función tendría que devolver el hash a la aplicación. No mejora frente a la igualdad sobre SHA-256 (decisión 2).
4. **Clave primaria `(fact_id, episode_id)` sin `on delete set null`, prohibiendo borrar episodios con hechos** — rechazada: hace imposible FR-14.
5. **Permitir `internal/store` en `internal/httpapi` e `internal/mcpserver`** — rechazada: abre la puerta a SQL y a consultas sin `Scope` desde los adaptadores; la verificación de credenciales cabe en el paquete raíz.
6. **Mover `audit_log` a una migración anterior para T-025** — rechazada: cambiar el orden de ejecución cuesta menos que romper la numeración de migraciones ya planificada.

## Notas de implementación
- Comando que vigila la decisión 2: `go test -tags=integration ./internal/store -run 'TestIsolation_|TestRLS_|TestAppRole_' -count=1`.
- `TestAppRole_NotOwnerNoBypass` (T-012) comprueba además que `mnemo_app` no tiene `CREATE` sobre `public` ni `TRUNCATE` o `TRIGGER` sobre ninguna tabla.
- Si la base de desarrollo se creó antes de cambiar `deploy/initdb/01-roles.sql`, hay que recrear el volumen (`docker compose down -v`), porque el script solo corre en una base vacía.

# mnemo — estrategia de pruebas

Estado: definida, sin código · Última revisión: 2026-11-19

Tres ejes que no se mezclan: **qué** se prueba (tipos), **cuánto** (cobertura por criticidad) y **a qué coste** (tamaño de la prueba). La evaluación con benchmarks es otra cosa y está en [evaluation-plan.md](evaluation-plan.md): las pruebas dicen si el código hace lo especificado; la evaluación dice si lo especificado sirve.

## 1. Tipos de prueba por componente

Tipos: TG1 unitaria · TG2 integración · TG3 componente/extremo a extremo de interfaz · TG4 contrato · TG5 humo · TG6 basada en propiedades.

| Componente | Criticidad | Tipos activos | Cobertura objetivo | Mutación | Pruebas primero |
|---|---|---|---|---|---|
| `internal/resolve` (máquina de estados de hechos) | **ALTA** — integridad de datos | TG1, TG6 | ≥ 90 % de ramas | Sí, antes de la versión 0.1 | Obligatorio |
| `internal/store` — aislamiento, RLS, purga | **ALTA** — autorización y datos personales | TG2 con PostgreSQL real | ≥ 90 % de ramas en el código de alcance y purga | No (es SQL y cableado) | Obligatorio |
| `internal/httpapi` — autenticación y ámbitos | **ALTA** | TG1, TG2, TG4 | ≥ 90 % de ramas en el middleware | No | Obligatorio |
| `internal/retrieve` (fusión, filtro temporal, ensamblado) | MEDIA-ALTA | TG1, TG6 | ≥ 80 % de ramas | Solo `FuseRRF` y `FilterAsOf` | Obligatorio |
| `internal/policy` (decaimiento, grupos de duplicados) | MEDIA-ALTA | TG1, TG6 | ≥ 80 % de ramas | No | Obligatorio |
| `internal/extract` (validación de salida, detector de instrucciones) | MEDIA-ALTA | TG1 con respuestas grabadas; TG2 con modelo falso | ≥ 80 % de ramas | No | Recomendado |
| `internal/store` — resto de consultas | MEDIA-ALTA | TG2 | ≥ 80 % de líneas | No | Recomendado |
| `internal/jobs` | MEDIA-ALTA | TG2 (concurrencia, reintentos, recuperación tras caída) | ≥ 80 % de ramas | No | Recomendado |
| `internal/provider` | MEDIA | TG1 con `httptest` (tiempos límite, reintentos, 429, JSON inválido) | ≥ 75 % de ramas | No | Recomendado |
| `internal/httpapi` — rutas | MEDIA-ALTA | TG2, TG4 | ≥ 80 % de líneas | No | Recomendado |
| `internal/mcpserver` | MEDIA-ALTA | TG2 con cliente del SDK en memoria; TG4 sobre los esquemas de herramientas | ≥ 80 % de líneas | No | Recomendado |
| Paquete raíz `mnemo` | MEDIA-ALTA | TG2; `Example` ejecutables | ≥ 80 % de líneas | No | Recomendado |
| `cmd/mnemo` | BAJA | TG5 | sin umbral | No | No |
| `eval/` (harness) | MEDIA — un error aquí falsea el titular | TG1 en métricas, percentiles, coste y división de datos; TG2 de la tubería con modelos falsos | ≥ 80 % de líneas en métricas e informes | No | Obligatorio para el cálculo de métricas |
| `tools/` | BAJA | TG1 mínima | sin umbral | No | No |

Decisiones por tipo:

- **TG3 apagado.** No hay interfaz de usuario.
- **TG4 encendido** para la API REST y las herramientas MCP: hay al menos tres clientes distintos (el harness en Python, clientes MCP de terceros, integradores) que evolucionan a otro ritmo. El contrato es `api/openapi.yaml`; una prueba compara las rutas, los códigos y el sobre de error implementados con el archivo, y otra fija los esquemas de las cuatro herramientas MCP en un archivo dorado.
- **TG6 encendido** donde hay invariantes estables sobre dominios grandes: la máquina de estados (a lo sumo un activo por cadena; ningún hecho desaparece; `valid_to ≥ valid_from`; aplicar dos veces el mismo veredicto no cambia nada), la fusión (invariante ante permutación de las listas; un documento en más listas nunca puntúa menos; determinista ante empates), el filtro temporal (un hecho aparece para `as_of` si y solo si `valid_from ≤ as_of < valid_to`), el decaimiento (monótono decreciente con la edad, creciente con los accesos). Se usa `testing/quick` y fuzzing nativo de Go: sin dependencias.
- **TG5:** `docker compose up` y una escritura y lectura reales.

**Realismo de las dependencias.** PostgreSQL siempre real (el de `docker compose`): RLS, índices únicos parciales y `SKIP LOCKED` no se pueden simular con honestidad. Modelos siempre falsos en las pruebas: `FakeEmbedder` determinista y un `ChatModel` guionizado que devuelve respuestas grabadas. Ninguna prueba llama a un proveedor real; eso es trabajo del harness.

**Omisiones deliberadas.** Sin pruebas de mutación fuera del núcleo de resolución y la fusión · sin pruebas de carga en CI (cuestan minutos y dan ruido; se ejecutan antes de cada versión) · sin pruebas sobre el texto de los prompts (su calidad se mide con el benchmark, no con aserciones) · sin pruebas "y": una prueba cuyo nombre lleva "y" se parte en dos.

## 2. Los cuatro niveles de validación

Comandos literales. De aquí sale la línea **Verificar** de cada tarea.

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

Variables que necesitan las pruebas de integración: `MNEMO_TEST_DATABASE_URL` (rol de aplicación) y `MNEMO_TEST_ADMIN_DATABASE_URL` (rol de migraciones). `docker-compose.yml` crea ambos roles.

## 3. Pruebas primero

Regla: en los componentes marcados como obligatorios no se escribe código de producción sin una prueba que falle antes por la razón esperada.

Orden dentro de una rebanada vertical: prueba de contrato → prueba de integración → prueba unitaria → implementación. En el plan, cuando una tarea es de criticidad ALTA, su comando Verificar es la prueba que se escribe primero.

Corrección de errores: primero la prueba que reproduce el error.

Señales de alarma a vigilar en las propias pruebas: más de la mitad de la prueba es montaje de dobles · la prueba pasa si se borra la aserción · se usa `time.Sleep` para esperar al trabajador (se usa `wait` o un canal) · se lee `time.Now()` dentro de una prueba de lógica temporal (se inyecta el reloj).

## 4. Diseño de casos

- **Máquina de estados de hechos:** una prueba por transición legal y una por transición prohibida, a partir del diagrama de [architecture.md](architecture.md#ciclo-de-vida-de-un-hecho). Tabla con nombre por fila.
- **Validez temporal:** valores límite en `valid_from − 1 ns`, `valid_from`, `valid_to − 1 ns`, `valid_to`; hechos sin `valid_from`; hechos sin `valid_to`.
- **Aislamiento:** matriz operación × recurso × (mismo inquilino, otro inquilino, credencial ligada a otro sujeto). Generada a partir de la tabla de rutas, para que una ruta nueva entre sola en la matriz.
- **Idempotencia:** misma clave y cuerpo; misma clave y otro cuerpo; dos peticiones simultáneas con la misma clave; reenvío tras la caducidad.
- **Degradación:** cada proveedor en cuatro modos: correcto, lento, error 5xx, respuesta inválida.
- **Límites:** 32.768 bytes y 32.769; 200 mensajes y 201; `max_tokens` 0, 1, 16.000 y 16.001.
- **Concurrencia:** dos sustituciones simultáneas de la misma cadena, repetida 20 veces; purga durante una extracción.

## 5. Cobertura

La cobertura es un cable trampa, no un objetivo. Se mide por ramas donde la herramienta lo permite y por líneas donde no (la herramienta estándar de Go mide sentencias; el umbral de "ramas" se aproxima con sentencias y se compensa con las pruebas de propiedades y, en el núcleo, con mutación).

`tools/covergate` lee `coverage.out` y falla si un paquete queda bajo su umbral o si la cobertura total cae más de 2 puntos respecto a la rama principal. Los archivos excluidos se listan en el propio programa con su motivo.

Mutación en `internal/resolve` e `internal/retrieve/fuse.go` antes de la versión 0.1. La herramienta se elige en T-018 (no investigué las opciones de mutación para Go; pregunta abierta Q-9). Si ninguna resulta razonable, se sustituye por una revisión manual de diez mutantes escritos a mano, y se dice.

Trampas a evitar: pruebas sin aserción, pruebas que solo comparan con una instantánea, puntos de entrada solo para pruebas en código de producción.

## 6. Trazabilidad

Cada criterio de aceptación del PRD tiene una prueba con el identificador del requisito en su nombre (`TestFR6_SupersedeClosesValidity`). El comando `go test ./... -tags=integration -list 'TestFR'` debe listar al menos una prueba por cada requisito de FR-1 a FR-23; FR-24 y FR-25 se cubren con el harness y con la prueba de humo. La comprobación de consistencia de cada hito cuenta esa lista.

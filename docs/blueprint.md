---
artifact: blueprint
version: 0.1.0
created: 2026-11-20
status: design
---

# mnemo — índice del diseño

**Estado: fase de diseño.** Sin código. Especificación, arquitectura, línea base y plan escritos; la comprobación de consistencia inicial está en [plan/README.md](plan/README.md#3-comprobación-de-consistencia) y tiene tres hallazgos HIGH que se resuelven antes de la tarea a la que bloquean (T-003, T-010, T-012). Primera tarea: T-001, pendiente del supuesto A-1.

mnemo es una capa de memoria de largo plazo para agentes LLM: biblioteca y servicio en Go, solo PostgreSQL, episodios inmutables y hechos bitemporales que se sustituyen, aislamiento por RLS, servidor MCP y un resultado reproducible en LongMemEval_S frente a tres líneas base.

## Documentos

| Documento | Qué decide o contiene |
|---|---|
| [README.md](../README.md) | Presentación en inglés: problema, qué hace, comparación con Mem0, Graphiti y Letta, tabla de resultados prevista, arquitectura en una figura |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Idioma, límites de módulos, convenciones de código y de pruebas, flujo por tarea, comandos literales, Definición de hecho |
| [LICENSE](../LICENSE) | MIT |
| [.gitignore](../.gitignore) | Fuera de git: secretos, `eval/data/`, `eval/runs/`, `eval/cache/`, volúmenes de PostgreSQL |
| [prd.md](prd.md) | Problema, usuarios, apetito (130 h, corte 2027-03-14, USD 60), métricas SM-1 … SM-6 y contramétricas, historias US-1 … US-10, alcance y orden de recorte, requisitos FR-1 … FR-25 con criterios de aceptación, riesgos R-1 … R-6, hitos, glosario, supuestos A-1 … A-10 |
| [architecture.md](architecture.md) | Invariantes I1-I5, características (la rectora: evaluabilidad), monolito modular, diagramas C4 y de secuencia, ciclo de vida de un hecho, módulos y reglas de dependencia, esquema SQL, idempotencia, API REST, herramientas MCP, API pública de Go, algoritmos, lo que no se construye, espina AD-1 … AD-6 |
| [adr/README.md](adr/README.md) | Índice de las decisiones de un solo sentido |
| [ADR-0001](adr/ADR-0001-postgresql-unico-almacen.md) | PostgreSQL 17 con pgvector, `pg_trgm` y texto completo como único almacén; sin índice vectorial en v1 |
| [ADR-0002](adr/ADR-0002-episodios-y-hechos-bitemporales.md) | Episodios inmutables y hechos con tiempo del mundo y del sistema; un conflicto sustituye, nunca borra |
| [ADR-0003](adr/ADR-0003-aislamiento-por-rls.md) | Credencial ligada a inquilino y opcionalmente a sujeto, `Scope` obligatorio en `internal/store`, RLS forzada con rol sin `BYPASSRLS` |
| [ADR-0004](adr/ADR-0004-benchmark-titular-y-protocolo.md) | LongMemEval_S como titular, LoCoMo como secundario, protocolo registrado antes de medir |
| [ADR-0005](adr/ADR-0005-extraccion-asincrona-en-postgres.md) | Escritura en dos fases con cola en PostgreSQL; `recall` sin modelo generativo |
| [ADR-0006](adr/ADR-0006-puertos-de-modelos-y-dimension.md) | Puertos HTTP compatibles con OpenAI, sin SDK de proveedor; Qwen3-Embedding-0.6B a 1.024 dimensiones |
| [security-baseline.md](security-baseline.md) | ASVS nivel 2 (nivel 3 en aislamiento y supresión), diagrama de flujo de datos, amenazas TH-1 … TH-19, casos de abuso AB-1 … AB-7, supresión, cadena de suministro, riesgos aceptados AR-1 … AR-7 |
| [quality-baseline.md](quality-baseline.md) | Funciones de aptitud con su comando, sensores, reglas de código, estabilidad por punto de integración, SLO, observabilidad, etapas de CI, deuda D-1 … D-7 |
| [testing-strategy.md](testing-strategy.md) | Tipos de prueba y cobertura por componente según criticidad, niveles de validación N1-N4, pruebas primero, diseño de casos, trazabilidad `TestFR<n>_` |
| [evaluation-plan.md](evaluation-plan.md) | Conjuntos, división `dev60` / 440 retenidas, brazos B0, B1, B2, M y ablaciones A1 … A9, protocolo, métricas, tabla de resultados, cifras de terceros, validación del juez, análisis de fallos, presupuesto, reproducción |
| [open-questions.md](open-questions.md) | Preguntas abiertas Q-1 … Q-24, seguimiento de supuestos A-1 … A-15 y anotaciones que deja cada tarea |
| [plan/README.md](plan/README.md) | Hitos M0 … M4 con fecha, grafo de dependencias, matriz FR → tarea → prueba, tarea → decisión, ADR → tareas, hallazgos F-1 … F-20 |
| [plan/tasks.md](plan/tasks.md) | Las 36 tareas en orden, cada una con requisito, decisión, archivos, interfaces, criterios, comando Verificar y riesgo; Definición de hecho |

Artefactos que crea una tarea y aún no existen: `api/openapi.yaml` (T-013), `eval/config/smoke.yaml` (T-006), `eval/config/protocol.yaml` (T-008), `docs/failure-analysis.md` (T-035), `CHANGELOG.md` y `.env.example` (T-036).

## Orden de lectura

1. [README.md](../README.md): qué es y qué promete medir.
2. [prd.md](prd.md): qué se construye, para quién y cómo se sabe si funcionó.
3. [architecture.md](architecture.md) y los [ADR](adr/README.md): cómo, y qué no se reabre.
4. [security-baseline.md](security-baseline.md), [quality-baseline.md](quality-baseline.md) y [testing-strategy.md](testing-strategy.md): qué tiene que pasar antes de dar algo por hecho.
5. [evaluation-plan.md](evaluation-plan.md): cómo se produce el número titular.
6. [plan/README.md](plan/README.md) y [plan/tasks.md](plan/tasks.md): en qué orden y con qué comprobación.
7. [open-questions.md](open-questions.md): lo que falta decidir.

Para implementar una tarea basta con [CONTRIBUTING.md](../CONTRIBUTING.md), el bloque de la tarea en [plan/tasks.md](plan/tasks.md) y los archivos de su "Leer antes". Para revisar el resultado sin leer todo: README → [evaluation-plan.md §6](evaluation-plan.md#6-tabla-de-resultados) → [§12](evaluation-plan.md#12-amenazas-a-la-validez) → [security-baseline.md §4](security-baseline.md#4-registro-de-amenazas).

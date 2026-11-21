# Registro de decisiones de arquitectura

Solo las decisiones de un solo sentido tienen ADR. Las reversibles están en la [espina de decisiones](../architecture.md#11-espina-de-decisiones).

| ID | Título | Estado | Fecha |
|---|---|---|---|
| [ADR-0001](ADR-0001-postgresql-unico-almacen.md) | PostgreSQL como único almacén | Aceptado | 2026-11-19 |
| [ADR-0002](ADR-0002-episodios-y-hechos-bitemporales.md) | Episodios inmutables y hechos bitemporales que se sustituyen | Aceptado | 2026-11-19 |
| [ADR-0003](ADR-0003-aislamiento-por-rls.md) | Aislamiento por inquilino con seguridad a nivel de fila y credenciales ligadas a sujeto | Aceptado | 2026-11-19 |
| [ADR-0004](ADR-0004-benchmark-titular-y-protocolo.md) | LongMemEval_S como benchmark titular y protocolo registrado | Aceptado | 2026-11-19 |
| [ADR-0005](ADR-0005-extraccion-asincrona-en-postgres.md) | Extracción asíncrona con cola en PostgreSQL y lectura sin modelo generativo | Aceptado | 2026-11-19 |
| [ADR-0006](ADR-0006-puertos-de-modelos-y-dimension.md) | Modelos tras puertos HTTP compatibles con OpenAI y dimensión de embedding fija en 1.024 | Aceptado | 2026-11-19 |
| [ADR-0007](ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md) | Operaciones sin inquilino conocido y ajustes del plan antes de T-001 (completa ADR-0003 y sustituye su nota sobre la comparación en tiempo constante) | Aceptado | 2026-11-21 |
| [ADR-0008](ADR-0008-dependencias-del-harness-de-evaluacion.md) | Dependencias de Python del harness de evaluación (completa la tabla de dependencias de security-baseline §8) | Aceptado | 2026-11-21 |

Un ADR aceptado no se edita: se sustituye por otro que lo referencia.

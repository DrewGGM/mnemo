# ADR-0008: Dependencias de Python del harness de evaluación

## Estado
Aceptado — 2026-11-21. Completa la tabla de dependencias de [security-baseline.md §8](../security-baseline.md#8-cadena-de-suministro), que solo trae las de Go.

## Reversibilidad
- **Tipo de puerta:** de dos sentidos. Tiene ADR porque la Definición de hecho ([CONTRIBUTING.md §7](../../CONTRIBUTING.md)) y security-baseline §8 lo piden para cada dependencia nueva, con nombre, licencia, mantenedores, última versión y coste de salida.
- **Coste de revertir:** horas para `ruff`, `pip-audit` y `uv_build`; un día para `pytest` o `pyyaml`.
- **Disparador de revisión:** una vulnerabilidad sin corregir que `pip-audit` bloquee; un cambio de licencia; o que un proyecto deje de publicar versiones.

## Contexto
T-006 crea el harness `mnemo-eval` en `eval/` ([architecture.md §5](../architecture.md#5-módulos)). El protocolo es un archivo YAML ([ADR-0004](ADR-0004-benchmark-titular-y-protocolo.md), notas de implementación) y la biblioteca estándar de Python no trae lector de YAML. El cliente HTTP usa `urllib` de la biblioteca estándar, sin dependencia. [ADR-0007](ADR-0007-operaciones-sin-inquilino-y-ajustes-del-plan.md), decisión 9, pide `pip-audit` como dependencia de desarrollo y `ruff` y `pytest` en el gancho y en el CI.

## Decisión
Versiones del 2026-11-21, fijadas en `eval/uv.lock` e instaladas en CI con `uv sync --frozen`:

| Dependencia | Grupo | Para qué | Licencia | Mantenedores | Versión fijada / última | Coste de salida |
|---|---|---|---|---|---|---|
| `pyyaml` | ejecución | Leer el protocolo (`load_protocol` en `pipeline.py`, con `yaml.safe_load`) | MIT | Proyecto YAML (organización `yaml` de GitHub); autor original, Kirill Simonov | 6.0.4 / 6.0.4 | Bajo: solo la usa `load_protocol`. Salir de ella es pasar el protocolo a TOML (`tomllib`, de la biblioteca estándar), lo que cambia su formato y obliga a sustituir ADR-0004 |
| `pytest` | desarrollo | Pruebas de `eval/tests` y marcador `integration` | MIT | pytest-dev | 9.2.0 / 9.2.0 | Medio: las pruebas usan `tmp_path`, `pytest.raises` y marcadores; pasarlas a `unittest` es reescribirlas |
| `ruff` | desarrollo | `ruff check` y `ruff format --check` | MIT | Astral Software Inc. | 0.17.6 / 0.17.6 | Bajo: solo configuración en `pyproject.toml` |
| `pip-audit` | desarrollo | Vulnerabilidades conocidas de las dependencias (compuerta de seguridad) | Apache-2.0 | PyPA (`pypa/pip-audit`), desarrollado por Trail of Bits | 2.10.1 / 2.10.1 | Bajo: un paso del CI |
| `uv_build` | construcción | Backend que construye `mnemo-eval` como paquete de espacio de nombres | MIT OR Apache-2.0 | Astral Software Inc. | `>=0.11.19,<0.12` / 0.13.9 | Bajo: se cambia por otro backend en `[build-system]` |

`uv_build` se acota a la rama 0.11, la de `uv` 0.11.19 que usa el CI, y se sube junto con `uv`.

`pyyaml` no tiene dependencias propias, así que en ejecución el harness no trae ninguna más. Las indirectas que arrastra `pip-audit` (`requests`, `cachecontrol`, `cyclonedx-python-lib` y otras, todas en `eval/uv.lock`) solo se instalan en desarrollo y en CI.

## Consecuencias
### Positivas
- Una sola dependencia en ejecución; el cliente HTTP no añade ninguna.
- `pip-audit` revisa en CI todo lo que fija `eval/uv.lock`.

### Negativas
- `pip-audit` multiplica las indirectas de desarrollo.

### Riesgos
- Un YAML de protocolo de terceros podría intentar construir objetos de Python → se lee con `yaml.safe_load`, nunca con `yaml.load`.

## Alternativas consideradas
1. **Protocolo en TOML con `tomllib`** — rechazada: ADR-0004 y [evaluation-plan.md](../evaluation-plan.md) fijan `eval/config/protocol.yaml`, y cambiarlo obliga a sustituir ADR-0004.
2. **Protocolo en JSON** — rechazada por lo mismo, y porque no admite comentarios, que `smoke.yaml` ya usa para explicar el protocolo.
3. **`requests` o `httpx` para el cliente** — rechazada: `urllib` basta para dos rutas POST con JSON.

## Notas de implementación
- `uv run --project eval pip-audit` el 2026-11-21: sin vulnerabilidades conocidas (omite `mnemo-eval`, que no está en PyPI).

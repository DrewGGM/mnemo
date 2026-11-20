# ADR-0005: Extracción asíncrona con cola en PostgreSQL y lectura sin modelo generativo

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** de un solo sentido para el contrato público: que `POST /v1/episodes` responda 202 con un trabajo, y que `recall` no dependa de un modelo generativo, son promesas que los clientes incorporan.
- **Coste de revertir:** unas 2 semanas de código y un cambio de versión de la API.
- **Disparador de revisión:** el retraso de la cola supera 5 minutos de forma sostenida; o el análisis de fallos muestra que reescribir la consulta con un modelo recuperaría más del 5 % de exactitud.

## Contexto
Extraer hechos cuesta una o dos llamadas a un modelo de chat, de segundos de duración y con fallos esperables. `recall` y la confirmación de escritura están en el camino de cada turno del agente. LongMemEval-V2 incorpora la latencia de consulta en su puntuación ([página](https://xiaowu0162.github.io/longmemeval-v2/)), y LongMemEval muestra que la expansión de consulta consciente del tiempo mejora el recall temporal pero necesita un modelo fuerte ([artículo](https://arxiv.org/html/2410.10813v2)): es una mejora real que aquí se sacrifica.

## Impulsores de la decisión
* **Imprescindible** — la escritura confirma en milisegundos y no se pierde si el modelo falla.
* **Imprescindible** — la lectura tiene latencia acotada y funciona con el proveedor de chat caído.
* **Imprescindible** — sin infraestructura adicional (ADR-0001).
* **Deseable** — el benchmark puede esperar a que la extracción termine, para ser determinista.

## Decisión
- **Escritura en dos fases.** `POST /v1/episodes` guarda los episodios y encola un trabajo `extract` en la **misma transacción**, y responde 202. Con `?wait=true` (o `WaitForExtraction` en Go) la llamada espera a que el trabajo termine, hasta 60 s.
- **Cola en PostgreSQL.** Tabla `jobs`; el trabajador corre en el mismo proceso, toma trabajos con `FOR UPDATE SKIP LOCKED`, reintenta hasta 5 veces con espera exponencial y deja el trabajo en `failed` con el error visible. Entrega al-menos-una-vez; la idempotencia la dan la clave de deduplicación y el índice único de hechos activos.
- **Lectura sin modelo generativo.** `recall` solo llama al embebedor (una vez) y, si se pide, al reranker. No hay reescritura de consulta, HyDE ni resumen en la lectura.

## Consecuencias
### Positivas
- Episodios y trabajo se confirman o fallan juntos: no hay episodio sin extracción pendiente ni trabajo huérfano.
- La memoria episódica está disponible de inmediato; la semántica, en segundos.
- Latencia de lectura predecible y coste de lectura casi nulo: solo las tokens de embeber la consulta.

### Negativas
- **Consistencia eventual visible:** un hecho recién dicho puede no estar todavía como hecho en la lectura siguiente. Se mitiga porque el episodio sí está y se recupera.
- Se renuncia a la expansión de consulta con modelo, que el artículo de LongMemEval mide como mejora en preguntas temporales.
- El cliente que necesite el resultado debe sondear o usar `wait`.

### Riesgos
- Un trabajo `running` cuyo proceso murió → `locked_at` con caducidad de 5 minutos; otro ciclo lo retoma.
- Ráfaga de escrituras que satura al proveedor → concurrencia del trabajador limitada (4 por defecto) y `run_after` con retroceso ante respuestas 429.
- La cola crece sin límite → limpieza diaria de trabajos terminados a los 30 días; métrica de profundidad y antigüedad de la cola.

### Puntos de sensibilidad
- La concurrencia del trabajador determina cuánto tarda la ingesta del benchmark.
- El tiempo límite de `wait` debe superar el de la llamada al modelo más reintentos.

### Puntos de compromiso
- **Latencia de escritura contra frescura de la memoria semántica.**
- **Latencia y coste de lectura contra exactitud en preguntas temporales**: se pierde la expansión de consulta y se compensa, en parte, con el filtro `as_of` y los intervalos de validez en el contexto.

## Alternativas consideradas
1. **Extracción síncrona en la petición** — rechazada: la escritura heredaría la latencia y la disponibilidad del proveedor de chat.
2. **Cola externa (Redis, NATS)** — rechazada: otro servicio que instalar, y escribir en dos sistemas sin transacción obligaría a un outbox.
3. **Extracción perezosa en la lectura** — rechazada: mueve el coste a la ruta sensible a latencia y hace la lectura no determinista.
4. **Reescritura de consulta con modelo, opcional por petición** — diferida, no rechazada: si el análisis de fallos la justifica, entra como opción explícita y como brazo de ablación, con ADR nuevo.

## Notas de implementación
- Función de aptitud: una prueba ejecuta `recall` con un `ChatModel` que hace fallar la prueba si se le llama.
- Prueba de recuperación: se mata el trabajador con un trabajo en curso y se comprueba que otro lo completa sin duplicar hechos.
- Métricas: `mnemo_jobs_queued`, `mnemo_jobs_oldest_age_seconds`, `mnemo_job_duration_seconds{kind,status}`.

# ADR-0001: PostgreSQL como único almacén

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** de un solo sentido. El esquema, las consultas, el aislamiento (ADR-0003) y la cola (ADR-0005) descansan sobre PostgreSQL.
- **Coste de revertir:** unas 4-6 semanas: reescribir `internal/store`, el aislamiento y la cola.
- **Disparador de revisión:** un sujeto supera 50.000 vectores y la latencia de `recall` incumple SM-4 aun con índice HNSW; o la ablación del grafo muestra que hacen falta recorridos de más de dos saltos.

## Contexto
mnemo necesita cuatro capacidades de almacenamiento: relacional con transacciones (sustitución atómica de hechos), similitud vectorial, búsqueda léxica y recorrido de un grafo pequeño de entidades. Una persona, sin presupuesto de infraestructura, con experiencia fuerte en PostgreSQL. El posicionamiento del proyecto es "un binario y una base". Los proyectos de referencia con grafo exigen una base aparte: Graphiti requiere Neo4j 5.26, FalkorDB 1.1.2 o Amazon Neptune ([README](https://github.com/getzep/graphiti)).

La memoria de un sujeto es pequeña: el volumen de diseño es de 5.000 hechos y 10.000 episodios. El grafo son entidades y hechos-arista del mismo sujeto, con recorridos de uno o dos saltos.

## Impulsores de la decisión
* **Imprescindible** — transacción única para insertar un hecho y cerrar el anterior.
* **Imprescindible** — arranque con un comando, sin servicios adicionales.
* **Imprescindible** — aislamiento por inquilino aplicado por la base (ADR-0003).
* **Deseable** — búsqueda híbrida sin proceso auxiliar.
* **Deseable** — coste cero de alojamiento para quien quiera probarlo (Neon ofrece 1 GB por proyecto con pgvector, [precios](https://neon.com/pricing)).

## Decisión
Todo el estado vive en PostgreSQL 17 con tres extensiones: `pgvector` (similitud), `pg_trgm` (nombres de entidades) y la búsqueda de texto completo nativa (`tsvector`, `ts_rank_cd`). El grafo se guarda como tablas `entities` y `facts` y se recorre con uniones y, si hace falta un segundo salto, una CTE recursiva acotada. No hay índice vectorial en v1: la búsqueda es exacta dentro de las filas del sujeto.

## Consecuencias
### Positivas
- Sustitución de hechos, cola de trabajos, idempotencia y consumo en la misma transacción.
- Un solo sistema que respaldar, purgar y aislar. La purga de un sujeto es un conjunto de `DELETE` en una transacción.
- Búsqueda exacta: sin el problema de recall de los índices aproximados con filtros selectivos.

### Negativas
- La búsqueda léxica **no es BM25**: `ts_rank_cd` no pondera por frecuencia inversa de documento. Se declara en el README y en la tabla de ablaciones.
- Sin lenguaje de consulta de grafos: los recorridos de más de dos saltos son incómodos y lentos.
- Escalar la búsqueda exacta más allá de decenas de miles de vectores por sujeto exige añadir un índice HNSW y revisar el filtrado.

### Riesgos
- Que la pata léxica rinda claramente peor que un BM25 real → se mide en la ablación A2; si la diferencia importa, se evalúa una extensión de BM25 como ADR nuevo.
- Que el supuesto A-7 (búsqueda exacta suficiente) sea falso → T-032 lo mide con 50.000 vectores.

### Puntos de sensibilidad
- La dimensión del vector (ADR-0006) determina el tamaño de fila y el coste de la búsqueda exacta.
- El índice `(tenant_id, subject_id, …)` es lo que convierte la búsqueda exacta en barata: toda consulta vectorial debe llevar ambos.

### Puntos de compromiso
- **Instalabilidad contra calidad de recuperación léxica**: se gana un único servicio y se pierde BM25.
- **Integridad contra escalabilidad**: transacciones fuertes en un nodo; no hay camino de escalado horizontal previsto.

## Alternativas consideradas
1. **PostgreSQL + base vectorial dedicada (Qdrant)** — rechazada: dos sistemas que mantener coherentes; escribir en dos fuentes sin transacción obliga a un outbox y a reconciliación; el volumen por sujeto no lo justifica.
2. **PostgreSQL + base de grafos (Neo4j, como Graphiti)** — rechazada: es justo lo que el proyecto quiere evitar; el grafo necesario es de uno o dos saltos.
3. **SQLite con extensión vectorial, embebido** — rechazada: instalación aún más simple, pero sin seguridad a nivel de fila ni concurrencia de escritores, y el aislamiento multi-inquilino es un requisito P0.
4. **Cloudflare D1 + Vectorize** — rechazada: Vectorize no tiene búsqueda léxica y limita `topK` a 50 con metadatos ([límites](https://developers.cloudflare.com/vectorize/platform/limits/)); no hay transacción entre ambos.

## Notas de implementación
- Imagen de desarrollo: `pgvector/pgvector:pg17` (supuesto A-9).
- Función de aptitud: la prueba de arranque exige que `/readyz` responda 200 con solo PostgreSQL levantado; el script de dependencias falla si `go.mod` gana un cliente de otra base de datos.
- Toda consulta vectorial lleva `tenant_id` y `subject_id` en el `WHERE`; lo comprueba una prueba que inspecciona el SQL de `internal/store`.

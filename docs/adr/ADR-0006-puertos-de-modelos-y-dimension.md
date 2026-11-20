# ADR-0006: Modelos tras puertos HTTP compatibles con OpenAI y dimensión de embedding fija en 1.024

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** mixta. El proveedor y el modelo de chat son de dos sentidos (configuración). La **dimensión del vector** y el **modelo de embeddings** son de un solo sentido en la práctica: cambiarlos invalida todos los vectores guardados.
- **Coste de revertir:** modelo de chat, minutos. Modelo o dimensión de embeddings: una migración de columna, re-embeber todo el contenido y repetir el benchmark.
- **Disparador de revisión:** el modelo de embeddings elegido deja de estar disponible en local o alojado; o una comparación propia muestra que otro modelo mejora el recall@10 en más de 5 puntos.

## Contexto
mnemo necesita tres capacidades externas: un modelo de chat con salida estructurada (extracción y veredictos), un modelo de embeddings y, opcionalmente, un reranker. Restricciones: pocas dependencias (SM-C5), presupuesto de USD 60, y reproducibilidad por terceros, que no deberían necesitar la misma cuenta de proveedor.

Datos disponibles:
- Qwen3-Embedding-0.6B: 1.024 dimensiones, Apache-2.0, 64,33 en MTEB multilingüe según su ficha (instantánea de junio de 2025, del propio proveedor) ([ficha](https://huggingface.co/Qwen/Qwen3-Embedding-8B)). Disponible alojado en Workers AI a USD 0,012 por millón de tokens ([precios](https://developers.cloudflare.com/workers-ai/platform/pricing/)).
- Reranker `bge-reranker-base` en Workers AI a USD 0,003 por millón de tokens ([precios](https://developers.cloudflare.com/workers-ai/platform/pricing/)).
- Gemini 3.1 Flash-Lite: USD 0,25 / 1,50 por millón de tokens, con capa gratuita y lotes a mitad de precio ([precios](https://aistudio.google.com/docs/pricing)). Sus límites gratuitos no se verificaron.
- El estado de los frameworks de LLM para Go solo se conoce por fuentes secundarias; ninguno se evaluó de primera mano.

## Impulsores de la decisión
* **Imprescindible** — el mismo modelo de embeddings en local y alojado, para que el benchmark se pueda reproducir sin pagar.
* **Imprescindible** — cero SDK de proveedor en el grafo de dependencias.
* **Imprescindible** — toda llamada con tiempo límite, reintentos acotados y consumo registrado.
* **Deseable** — cambiar de proveedor de chat con una variable de entorno.

## Decisión
1. **Tres interfaces pequeñas** en el paquete raíz: `ChatModel`, `Embedder`, `Reranker`. Las implementa `internal/provider` con `net/http` de la biblioteca estándar.
2. **Chat y embeddings hablan el protocolo compatible con OpenAI** (`/chat/completions` con salida JSON con esquema; `/embeddings`). Cualquier proveedor o servidor local que lo ofrezca sirve sin código nuevo.
3. **El reranker** usa un contrato mínimo propio (`POST {query, documents[]} → {scores[]}`) con un adaptador para Workers AI.
4. **Embedding fijo: Qwen3-Embedding-0.6B a 1.024 dimensiones**, columna `vector(1024)`. El nombre del modelo se guarda junto a la configuración de la instancia y el arranque falla si no coincide con el de los vectores existentes.
5. **Sin framework de LLM.** Prompts como archivos de texto versionados y embebidos en el binario.
6. Valores por defecto del benchmark: extractor y lector Gemini 3.1 Flash-Lite; embeddings Qwen3-Embedding-0.6B en local. Son configuración, no código (supuestos A-3, A-4, A-5).

## Consecuencias
### Positivas
- Sin dependencias de proveedor; el grafo de dependencias sigue siendo de un vistazo.
- Reproducible con modelos locales: quien no quiera pagar puede correr todo salvo el juez.
- La política de tiempos límite, reintentos y registro de consumo vive en un solo lugar.

### Negativas
- "Compatible con OpenAI" es un mínimo común: las funciones propias de cada proveedor (caché de prompts, lotes) quedan fuera del adaptador genérico. El modo por lotes, que abarataría la ingesta, exigiría un adaptador específico.
- Una única dimensión: no se pueden mezclar modelos por inquilino.
- Las cifras de calidad del modelo de embeddings vienen de la ficha del proveedor, no de una medición propia sobre conversaciones.

### Riesgos
- La compatibilidad del endpoint de un proveedor es parcial (salida estructurada distinta) → T-007 lo verifica contra el endpoint real antes de depender de él; alternativa: un adaptador específico de unas 100 líneas.
- El modelo de embeddings rinde mal en conversación informal → el brazo B2 lo usa también, así que la comparación interna sigue siendo justa; el número absoluto es lo que sufre.

### Puntos de sensibilidad
- La dimensión determina el tamaño de fila y el coste de la búsqueda exacta (ADR-0001).
- El modelo de chat determina la calidad de la extracción, que es lo que más mueve SM-1.

### Puntos de compromiso
- **Portabilidad contra aprovechar funciones del proveedor** (lotes, caché).
- **Simplicidad contra flexibilidad**: una dimensión para todos.

## Alternativas consideradas
1. **SDK oficial de un proveedor** — rechazada: ata el proyecto a un proveedor y añade dependencias; no se investigaron los SDK de Go de los proveedores.
2. **Framework de LLM para Go** — rechazada: la evidencia sobre su madurez es de segunda mano, y el proyecto usa tres llamadas HTTP.
3. **`text-embedding-3-small` (1.536 dimensiones)** — rechazada: no se puede ejecutar en local; el precio de USD 0,02 por millón de tokens solo se conoce por fuentes secundarias.
4. **`bge-m3`** — rechazada: 59,56 en MTEB multilingüe frente a 64,33 de Qwen3-Embedding-0.6B en la misma ficha, al mismo precio alojado.
5. **Dimensión configurable por instancia con columna sin tipo fijo** — rechazada: complica índices y pruebas para un caso que no existe todavía.

## Notas de implementación
- Tiempos límite: embeddings 5 s, chat 30 s, reranker 1,5 s. Reintentos: 3 con retroceso exponencial y fluctuación, solo ante 429, 5xx y errores de red. Cortacircuitos solo en el reranker (5 fallos seguidos lo abren 30 s).
- Función de aptitud: el script de dependencias falla si `go.mod` supera 6 dependencias directas; una prueba de arranque falla si `Embedder.Dim()` no es 1.024 o si el modelo configurado no coincide con el registrado.
- `FakeEmbedder` determinista (bolsa de palabras con hash a 1.024 dimensiones) para que las pruebas y el CI no necesiten red.

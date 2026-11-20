# ADR-0003: Aislamiento por inquilino con seguridad a nivel de fila y credenciales ligadas a sujeto

## Estado
Aceptado — 2026-11-19

## Reversibilidad
- **Tipo de puerta:** de un solo sentido. Retirar o cambiar el modelo de aislamiento con datos reales dentro es una migración de riesgo alto.
- **Coste de revertir:** unas 3 semanas, más una auditoría completa de consultas.
- **Disparador de revisión:** un cliente exige aislamiento físico (base por inquilino), o la sobrecarga de RLS medida en `recall` supera el 15 % del p95.

## Contexto
Una instancia sirve a varios inquilinos y la memoria contiene datos personales. La fuga entre inquilinos es el fallo más grave posible y, en recuperación con vectores, ocurre por omisión: basta una consulta sin filtro. Además, dentro de un inquilino cada sujeto es una persona distinta, y el servidor MCP pone un modelo de lenguaje a elegir los argumentos de las llamadas: si el sujeto fuera un argumento, un texto inyectado podría pedir la memoria de otra persona.

Los servidores de referencia de Graphiti (MCP y REST) se publican sin autenticación ni autorización, según su propio README ([Graphiti](https://github.com/getzep/graphiti)). Aquí el aislamiento es parte del producto.

## Impulsores de la decisión
* **Imprescindible** — una consulta que olvide el filtro de inquilino no debe devolver filas ajenas.
* **Imprescindible** — el sujeto no puede ser elegido por un modelo de lenguaje.
* **Imprescindible** — purga y exportación por sujeto sencillas.
* **Deseable** — una sola base y un solo esquema (ADR-0001).

## Decisión
Tres capas, de fuera hacia dentro:

1. **Credencial.** Clave de API de alta entropía, guardada como SHA-256, ligada a un inquilino, con ámbitos (`read`, `write`, `admin`) y, opcionalmente, a **un único sujeto**. Una credencial ligada a sujeto rechaza con 403 cualquier otro. Las credenciales para MCP son siempre ligadas a sujeto, y las herramientas MCP no tienen parámetro de sujeto ni de inquilino.
2. **Aplicación.** `internal/store` es el único paquete que ejecuta SQL y todas sus funciones reciben un `Scope` (inquilino, sujeto); no existe una función que consulte sin él.
3. **Base de datos.** Columna `tenant_id` en todas las tablas de datos, con `ENABLE` y `FORCE ROW LEVEL SECURITY` y la política `tenant_id = current_setting('app.tenant_id')::uuid`. La aplicación se conecta con un rol sin `BYPASSRLS` y sin propiedad de las tablas, y fija `app.tenant_id` con `SET LOCAL` al abrir cada transacción. Sin ese ajuste, `current_setting` falla y la consulta no devuelve nada: falla cerrado.

Un identificador de otro inquilino se comporta como inexistente (404), para no revelar que existe.

## Consecuencias
### Positivas
- Defensa en profundidad: un error en la capa 2 lo detiene la capa 3.
- El aislamiento se prueba de forma genérica: una prueba recorre las tablas con `tenant_id` y verifica política y resultado.
- El flujo "entrada no confiable → herramienta → memoria de otra persona" queda cortado por diseño, no por instrucción en un prompt.

### Negativas
- Toda operación debe ir en una transacción para que `SET LOCAL` aplique; no se puede usar el pool con consultas sueltas.
- Las migraciones y los subcomandos de administración necesitan un segundo rol.
- Una credencial por sujeto es más gestión para quien integra MCP con muchos usuarios.

### Riesgos
- Reutilización de conexiones del pool con un `app.tenant_id` heredado → se usa `SET LOCAL`, que muere con la transacción, y una prueba lo verifica alternando inquilinos sobre una única conexión.
- El rol de la aplicación termina siendo dueño de las tablas por una migración descuidada → una prueba de arranque falla si el rol actual es dueño de alguna tabla o tiene `BYPASSRLS`.

### Puntos de sensibilidad
- Cualquier tabla nueva con datos de inquilino necesita política; la prueba genérica falla si una tabla con `tenant_id` no la tiene.
- El orden de columnas de los índices empieza por `tenant_id`.

### Puntos de compromiso
- **Aislamiento contra rendimiento**: cada consulta paga la evaluación de la política y una transacción.
- **Seguridad contra ergonomía**: el cliente MCP no puede cambiar de sujeto sin cambiar de credencial.

## Alternativas consideradas
1. **Solo filtro `WHERE tenant_id` en el código** — rechazada: una omisión es una fuga, y es el error más probable.
2. **Esquema o base por inquilino** — rechazada: aislamiento más fuerte, pero migraciones por inquilino, pools separados y complejidad operativa que una persona no puede sostener. Queda como opción si aparece un cliente que lo exija.
3. **Sujeto como argumento de las herramientas MCP** — rechazada: convierte una inyección de prompt en lectura de memoria ajena.
4. **JWT con reclamaciones de inquilino y sujeto** — diferida: resuelve mejor el caso de muchos usuarios finales, pero exige emisor, rotación y validación; no hace falta para el autoalojado.

## Notas de implementación
- Formato de clave: `mnemo_<prefijo de 8>_<32 bytes en base64url>`; se muestra una sola vez. El prefijo permite identificarla en registros sin exponerla. Comparación de hashes en tiempo constante.
- Funciones de aptitud: `TestIsolation_AllTables` (para cada tabla con `tenant_id`: inserta como A, consulta como B con SQL sin `WHERE`, exige 0 filas), `TestRLS_ForcedOnAllTenantTables`, `TestAppRole_NotOwnerNoBypass`.
- Los subcomandos `mnemo tenant create` y `mnemo key create --subject` son la única forma de crear inquilinos y credenciales; no hay endpoint HTTP para ello.

# Node.js Audit Rules

Reglas específicas para proyectos Node.js.

## 1. Arquitectura

Revisar:

* Separación entre rutas, controladores, servicios y acceso a datos.
* Lógica de negocio dentro de rutas.
* Acceso directo a base de datos desde múltiples capas.
* Dependencias circulares.
* Módulos con demasiadas responsabilidades.
* Acoplamiento innecesario.

No imponer una arquitectura concreta si la existente es coherente.

---

## 2. API

Cuando exista una API HTTP, revisar:

* Validación de parámetros.
* Validación del body.
* Validación de query params.
* Códigos HTTP utilizados incorrectamente.
* Respuestas inconsistentes.
* Manejo incorrecto de errores.
* Exposición innecesaria de información.
* Endpoints duplicados o inconsistentes.

---

## 3. Middleware

Detectar:

* Middleware con demasiadas responsabilidades.
* Middleware aplicado globalmente cuando debería ser específico.
* Orden incorrecto de middleware.
* Middleware duplicado.
* Middleware que modifica objetos de request/response de forma inesperada.

---

## 4. Autenticación y autorización

Revisar:

* Verificación de autenticación.
* Control de permisos.
* Validación de identidad.
* Acceso a recursos de otros usuarios o locales.
* Información sensible devuelta por endpoints.
* Diferencia entre autenticación y autorización.

Especial atención a endpoints donde el identificador del recurso proviene directamente del cliente.

---

## 5. Errores

Revisar:

* Errores no capturados.
* `try/catch` excesivos.
* Errores enviados directamente al cliente.
* Stack traces expuestos en producción.
* Errores convertidos incorrectamente.
* Diferentes formatos de error para situaciones equivalentes.

Preferir una estrategia consistente de manejo de errores.

---

## 6. Async / Await

Detectar:

* Promesas sin manejar.
* Operaciones independientes ejecutadas secuencialmente.
* Callbacks innecesarios.
* `await` innecesarios.
* Operaciones asíncronas dentro de `forEach`.
* Errores asincrónicos que no llegan al manejador correspondiente.

---

## 7. Base de datos

Cuando Node.js interactúe con una base de datos, revisar:

* Consultas repetitivas.
* N+1 queries.
* Consultas innecesarias.
* Falta de transacciones cuando varias operaciones deben ser atómicas.
* Conexiones mal gestionadas.
* Queries construidas de forma insegura.
* Datos obtenidos que no son utilizados.

No recomendar transacciones indiscriminadamente; evaluar los límites reales de consistencia.

---

## 8. Transacciones

Detectar operaciones donde:

```text id="mquy9a"
Operación A
    ↓
Operación B
    ↓
Operación C
```

deban considerarse una única unidad atómica.

Evaluar qué ocurre si una operación intermedia falla.

---

## 9. Validación

Revisar validación de:

* `req.body`
* `req.params`
* `req.query`
* Headers
* Datos provenientes de servicios externos.

La validación del cliente no debe considerarse suficiente para proteger el backend.

---

## 10. Variables de entorno

Detectar:

* Secrets incluidos en el código.
* Variables obligatorias sin validar.
* Valores sensibles registrados en logs.
* Configuración duplicada.
* Diferencias inesperadas entre entornos.

Cuando sea posible, validar la configuración durante el arranque de la aplicación.

---

## 11. Logs

Revisar:

* Logs excesivos.
* Información sensible.
* Passwords.
* Tokens.
* Datos personales innecesarios.
* Errores sin contexto.
* `console.log` utilizados como mecanismo permanente de logging cuando exista una infraestructura de logs.

No considerar `console.log` automáticamente incorrecto durante desarrollo.

---

## 12. Seguridad

Buscar:

* SQL Injection.
* Command Injection.
* Path Traversal.
* SSRF.
* XSS cuando el servidor genere contenido.
* Exposición de secrets.
* CORS excesivamente permisivo.
* Falta de rate limiting cuando el contexto lo requiera.
* Headers de seguridad ausentes cuando sean relevantes.
* Deserialización insegura.

Reportar únicamente vulnerabilidades respaldadas por evidencia.

---

## 13. Dependencias

Revisar:

* Dependencias no utilizadas.
* Dependencias duplicadas.
* Paquetes innecesarios.
* Dependencias obsoletas cuando pueda verificarse.
* Scripts potencialmente peligrosos.
* Diferencias entre `dependencies` y `devDependencies`.

Revisar:

```text id="5stcfr"
package.json
package-lock.json
yarn.lock
pnpm-lock.yaml
```

según corresponda.

---

## 14. Procesos y recursos

Detectar:

* Conexiones que no se cierran.
* Timers que permanecen activos innecesariamente.
* Event listeners acumulados.
* Streams sin liberar.
* Procesos secundarios mal gestionados.
* Recursos externos sin cleanup.

---

## 15. Configuración de producción

Cuando sea relevante revisar:

* Manejo de errores de producción.
* Variables de entorno.
* CORS.
* Logging.
* Seguridad HTTP.
* Timeouts.
* Límites de payload.
* Manejo de señales del proceso.
* Graceful shutdown.

---

## 16. Rendimiento

Buscar:

* Operaciones bloqueantes en el event loop.
* Procesamiento CPU-intensive ejecutado directamente en el proceso principal.
* Consultas innecesarias.
* Serialización/deserialización excesiva.
* Payloads innecesariamente grandes.
* Operaciones repetidas.

No recomendar clustering, workers o microservicios sin una necesidad demostrable.

---

## 17. Estructura de respuestas

Si el proyecto expone una API, revisar que exista consistencia en:

```text id="c7s0wq"
Éxito
Errores
Paginación
Validación
Campos opcionales
Códigos HTTP
```

No modificar el contrato existente únicamente por preferencia.

---

## 18. Principio específico de Node.js

Priorizar problemas relacionados con:

* Seguridad de APIs.
* Autenticación y autorización.
* Manejo de errores.
* Operaciones bloqueantes.
* Race conditions.
* Transacciones.
* Consultas ineficientes.
* Fugas de recursos.
* Configuración insegura.
* Exposición de información.

La auditoría debe centrarse en problemas que puedan afectar realmente el comportamiento, seguridad, rendimiento o mantenibilidad del backend.


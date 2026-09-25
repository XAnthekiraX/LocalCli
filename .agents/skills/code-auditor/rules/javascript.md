# JavaScript Audit Rules

Reglas específicas para proyectos JavaScript.

## 1. Uso del lenguaje

Revisar:

* Uso innecesario de `var`.
* Uso incorrecto de `let` y `const`.
* Comparaciones débiles (`==`, `!=`) cuando no están justificadas.
* Conversiones implícitas que puedan generar errores.
* Uso innecesario de `eval`.
* Mutaciones innecesarias.
* Variables declaradas con alcance mayor al necesario.

---

## 2. Funciones

Detectar:

* Funciones excesivamente grandes.
* Demasiados parámetros.
* Funciones con múltiples responsabilidades.
* Callbacks profundamente anidados.
* Funciones que mezclan lógica síncrona y asíncrona de forma problemática.

---

## 3. Async / Await

Revisar:

* Promesas sin `await` cuando corresponde.
* `await` innecesarios.
* Uso incorrecto de `Promise`.
* Operaciones independientes ejecutadas secuencialmente.
* Falta de manejo de errores.
* Mezcla innecesaria de callbacks, Promises y `async/await`.

Ejemplo conceptual:

```text
Operación A ─┐
             ├─ pueden ejecutarse en paralelo
Operación B ─┘
```

Si el código las ejecuta secuencialmente sin necesidad, reportarlo cuando exista impacto real.

---

## 4. Promesas

Detectar:

* Promesas rechazadas sin manejar.
* Cadenas innecesariamente complejas.
* `.then()` mezclado innecesariamente con `async/await`.
* Creación manual de Promises cuando no es necesaria.
* `Promise.all` utilizado cuando las operaciones tienen dependencias entre sí.

---

## 5. Arrays y objetos

Revisar:

* Uso innecesario de `map`, `filter` o `reduce`.
* Múltiples recorridos innecesarios cuando pueden evitarse sin perder claridad.
* Mutaciones inesperadas.
* Copias innecesarias de objetos grandes.
* Uso incorrecto de `forEach` con operaciones asíncronas.

Especial atención a:

```js
array.forEach(async () => {})
```

cuando el código espera que las operaciones sean esperadas.

---

## 6. Manejo de `null` y `undefined`

Detectar:

* Acceso potencial a propiedades inexistentes.
* Comprobaciones inconsistentes.
* Uso incorrecto de `||` cuando `0`, `false` o `""` son valores válidos.
* Uso innecesario o incorrecto de optional chaining (`?.`).
* Uso incorrecto de nullish coalescing (`??`).

---

## 7. Scope y closures

Revisar:

* Variables capturadas accidentalmente por closures.
* Dependencias ocultas.
* Variables globales.
* Estado compartido innecesariamente.
* Closures que mantienen referencias durante más tiempo del necesario.

---

## 8. Módulos

Detectar:

* Imports innecesarios.
* Exports sin uso.
* Dependencias circulares.
* Módulos excesivamente grandes.
* Módulos con responsabilidades no relacionadas.
* Mezcla innecesaria de CommonJS y ES Modules.

---

## 9. Manejo de errores

Revisar:

* `try/catch` innecesarios.
* `catch` que ocultan errores.
* Lanzamiento de valores que no sean errores cuando dificulta el diagnóstico.
* Pérdida del contexto original del error.
* Errores asincrónicos sin manejar.

Preferir errores que permitan identificar claramente:

```text
Qué ocurrió
Dónde ocurrió
Por qué ocurrió
```

---

## 10. Coerción y tipos dinámicos

Detectar código que dependa accidentalmente de coerción de tipos.

Especial atención a:

```js
"" + value
Number(value)
Boolean(value)
!!value
value == other
```

No considerar estas operaciones incorrectas por sí mismas; evaluar su contexto.

---

## 11. Variables y constantes

Buscar:

* Variables que nunca cambian y deberían ser `const`.
* Variables reutilizadas con significados diferentes.
* Declaraciones demasiado alejadas de su uso.
* Constantes duplicadas.
* Valores mágicos repetidos.

---

## 12. Patrones problemáticos

Revisar patrones conocidos como:

* `forEach(async ...)`.
* `new Promise(async ...)`.
* `await` dentro de ciclos cuando existe una alternativa segura.
* Callbacks anidados innecesariamente.
* Estado global mutable.
* Modificación inesperada de objetos recibidos como argumentos.

---

## 13. Compatibilidad

Si el proyecto define una versión concreta de Node.js, navegador o runtime:

* Respetar las capacidades soportadas.
* No recomendar APIs incompatibles.
* Detectar APIs obsoletas cuando sea relevante.

Verificar primero `package.json`, configuración del proyecto y herramientas de build.

---

## 14. Dependencias y paquetes

Revisar:

* Paquetes importados pero no utilizados.
* Dependencias declaradas en la sección incorrecta.
* Librerías utilizadas para tareas que JavaScript ya resuelve adecuadamente.
* Dependencias duplicadas para resolver el mismo problema.

No recomendar reemplazar una dependencia únicamente por preferencia personal.

---

## 15. Principio específico de JavaScript

Priorizar problemas que puedan producir:

* Bugs por coerción.
* Errores asincrónicos.
* Condiciones de carrera.
* Mutaciones inesperadas.
* Problemas de scope.
* Memory leaks.
* Código difícil de mantener.

No convertir preferencias de estilo JavaScript en hallazgos de alta prioridad.


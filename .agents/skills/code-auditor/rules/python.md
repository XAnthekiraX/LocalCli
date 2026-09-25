# Python Audit Rules

Reglas específicas para proyectos Python.

## 1. Código Python

Revisar:

* Uso innecesario de variables globales.
* Funciones excesivamente grandes.
* Mutaciones inesperadas.
* Código duplicado.
* Comprensiones innecesariamente complejas.
* Uso incorrecto de valores por defecto mutables.
* Imports innecesarios.

---

## 2. Tipado

Cuando el proyecto utilice type hints, revisar:

* Tipos ausentes en APIs importantes.
* `Any` innecesario.
* Tipos incorrectos.
* `Optional` utilizado incorrectamente.
* Tipos duplicados o inconsistentes.

No exigir type hints en todo el proyecto si no forman parte de su estrategia.

---

## 3. Excepciones

Detectar:

* `except Exception` excesivamente amplio.
* `except:` sin especificar excepción.
* Excepciones ignoradas.
* `pass` utilizado para ocultar errores.
* Excepciones utilizadas para controlar flujo normal sin justificación.
* Pérdida del contexto original.

---

## 4. Recursos

Revisar correctamente el manejo de:

* Archivos.
* Conexiones.
* Streams.
* Locks.
* Recursos externos.

Preferir mecanismos como context managers cuando correspondan.

---

## 5. Async

Si el proyecto utiliza `asyncio`, revisar:

* Funciones async que realizan operaciones bloqueantes.
* `await` innecesarios.
* Tareas creadas pero no esperadas.
* Operaciones independientes ejecutadas secuencialmente.
* Mezcla incorrecta de código síncrono y asíncrono.

---

## 6. Clases

Detectar:

* Clases con demasiadas responsabilidades.
* Herencia innecesaria.
* Jerarquías excesivamente complejas.
* Métodos excesivamente grandes.
* Estado mutable difícil de controlar.

No recomendar clases cuando una función o módulo sea suficiente.

---

## 7. Decoradores

Revisar:

* Decoradores con efectos secundarios inesperados.
* Decoradores excesivamente complejos.
* Pérdida de metadata de funciones cuando sea relevante.
* Decoradores utilizados únicamente para ocultar lógica sencilla.

---

## 8. Iteraciones y colecciones

Detectar:

* Recorridos innecesarios.
* Creación de estructuras intermedias innecesarias.
* Uso incorrecto de generadores.
* Comprensiones difíciles de leer.
* Mutación de colecciones durante la iteración.

Priorizar claridad cuando la optimización no tenga impacto significativo.

---

## 9. Dependencias

Revisar:

* Imports no utilizados.
* Dependencias no utilizadas.
* Dependencias duplicadas.
* Librerías innecesarias.
* Dependencias utilizadas para resolver problemas simples sin justificación.

Verificar archivos como:

```text
requirements.txt
pyproject.toml
Pipfile
poetry.lock
uv.lock
```

cuando existan.

---

## 10. Configuración

Revisar:

* Secrets dentro del código.
* Configuración duplicada.
* Valores específicos del entorno.
* Variables de entorno utilizadas incorrectamente.
* Configuración dispersa innecesariamente.

---

## 11. Rendimiento

Buscar problemas reales como:

* Consultas repetitivas.
* Operaciones costosas dentro de loops.
* Lectura innecesaria de archivos.
* Creación excesiva de objetos.
* Uso incorrecto de estructuras de datos.

No recomendar micro-optimizaciones sin impacto demostrable.

---

## 12. Python idiomático

Detectar construcciones innecesariamente complejas cuando exista una alternativa Python clara y más mantenible.

Ejemplos:

* Código repetitivo.
* Condicionales innecesarios.
* Manipulación manual de estructuras que Python resuelve directamente.
* Implementaciones propias de funcionalidades estándar.

No considerar una solución incorrecta simplemente por no ser la alternativa más idiomática.

---

## 13. Mutabilidad

Revisar:

* Estado global mutable.
* Argumentos modificados dentro de funciones.
* Estructuras compartidas modificadas inesperadamente.
* Valores por defecto mutables.

Especial atención a:

```python
def funcion(items=[]):
    ...
```

cuando el estado pueda persistir entre llamadas.

---

## 14. Seguridad

Buscar:

* `eval`.
* `exec`.
* Deserialización insegura.
* SQL construido mediante concatenación.
* Ejecución de comandos externos insegura.
* Secrets expuestos.
* Validación insuficiente de entradas externas.

---

## 15. Principio específico de Python

Priorizar problemas relacionados con:

* Excepciones ocultas.
* Mutabilidad inesperada.
* Código async incorrecto.
* Recursos sin liberar.
* Tipado inseguro cuando el proyecto utiliza typing.
* Dependencias innecesarias.
* Problemas reales de rendimiento o seguridad.

La auditoría debe respetar el estilo y arquitectura existentes del proyecto en lugar de imponer una única forma de escribir Python.


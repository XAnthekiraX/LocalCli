# General Code Audit Rules

Reglas aplicables independientemente del lenguaje o framework.

## 1. Código muerto

Detectar:

* Funciones nunca utilizadas.
* Variables o constantes sin uso.
* Imports innecesarios.
* Archivos sin referencias evidentes.
* Código comentado que ya no tiene propósito claro.
* Funcionalidades abandonadas.
* Dependencias instaladas pero no utilizadas.

**Nota:** No considerar código potencialmente utilizado dinámicamente (reflection, metaprogramación, imports condicionales, usos generados) como muerto sin evidencia suficiente. En caso de duda, clasificar como `[INVESTIGATE]`.

---

## 2. Duplicación

Buscar:

* Lógica repetida.
* Funciones prácticamente iguales.
* Validaciones duplicadas.
* Transformaciones repetidas.
* Consultas repetidas.
* Constantes duplicadas.
* Implementaciones diferentes del mismo comportamiento.

**Principio:** Evaluar si realmente conviene abstraer antes de recomendar una refactorización. No recomendar abstraer duplicaciones triviales si la abstracción aumenta innecesariamente la complejidad.

---

## 3. Complejidad

Detectar:

* Funciones excesivamente grandes.
* Condicionales profundamente anidados.
* Flujo difícil de seguir.
* Demasiadas responsabilidades en una misma función.
* Dependencias innecesariamente complejas.
* Soluciones más complejas de lo necesario.

**Principio:** No dividir código únicamente por cantidad de líneas. Justificar la división por claridad o reducción de responsabilidades reales.

---

## 4. Responsabilidades

Verificar que cada módulo, clase o función tenga responsabilidades coherentes y bien delimitadas.

Detectar:

* Funciones que hacen demasiadas cosas.
* Lógica de negocio mezclada con infraestructura.
* Acceso a datos mezclado con presentación.
* Validación mezclada con transformación y persistencia.
* Módulos con responsabilidades no relacionadas.

---

## 5. Nombres

Revisar:

* Variables.
* Funciones.
* Clases.
* Módulos.
* Parámetros.
* Constantes.

Detectar nombres:

* Ambiguos.
* Engañosos.
* Demasiado genéricos.
* Inconsistentes.
* Que no representan correctamente su propósito.

**Principio:** No considerar un nombre problemático únicamente por preferencia personal. Evaluar por claridad y consistencia.

---

## 6. Manejo de errores

Buscar:

* Errores ignorados.
* `catch` vacíos (sin justificación).
* Excepciones ocultadas.
* Mensajes de error poco útiles.
* Errores convertidos incorrectamente.
* Falta de manejo en operaciones críticas.
* Tratamiento inconsistente de errores entre flujos similares.

**Principio:** Evaluar si el error debe propagarse, transformarse o manejarse localmente según el contexto.

---

## 7. Validación

Verificar que los datos externos sean validados cuando corresponda.

Considerar:

* Inputs del usuario.
* Parámetros de API.
* Datos provenientes de archivos.
* Datos provenientes de servicios externos.
* Configuración.
* Datos persistidos que requieren validación adicional.

Distinguir entre validación de formato, reglas de negocio y restricciones de persistencia. No exigir validación excesiva donde no exista riesgo demostrable.

---

## 8. Dependencias

Revisar:

* Dependencias innecesarias.
* Dependencias duplicadas.
* Dependencias utilizadas incorrectamente.
* Dependencias excesivas para tareas simples.
* Versiones o configuraciones potencialmente problemáticas cuando puedan verificarse.

**Principio:** No recomendar eliminar una dependencia sin comprobar su uso real.

---

## 9. Configuración

Detectar:

* Configuración duplicada.
* Valores sensibles dentro del código.
* Configuración innecesariamente dispersa.
* Valores mágicos.
* Configuración específica del entorno mezclada con código.
* Variables de entorno utilizadas incorrectamente.

---

## 10. Seguridad

Buscar problemas generales relacionados con:

* Datos sensibles.
* Validación de entradas.
* Control de acceso.
* Exposición de información.
* Secrets.
* Logs.
* Inyección.
* Configuraciones inseguras.

**Principio:** Solo reportar vulnerabilidades cuando exista evidencia suficiente. No asumir explotabilidad sin fundamento.

---

## 11. Rendimiento

Buscar únicamente problemas con impacto razonablemente demostrable.

Considerar:

* Trabajo innecesario.
* Operaciones repetidas.
* Consultas innecesarias.
* Procesamiento de grandes cantidades de datos.
* Uso ineficiente de recursos.
* Operaciones síncronas costosas cuando deberían ser asíncronas.

**Principio:** No optimizar prematuramente sin evidencia de impacto.

---

## 12. Consistencia

Detectar implementaciones inconsistentes del mismo concepto.

Ejemplos:

* Diferentes convenciones de nombres.
* Diferentes formatos de respuesta.
* Diferentes mecanismos para manejar errores similares.
* Diferentes patrones para acceder al mismo tipo de recurso.
* Convenciones aplicadas solo parcialmente.

---

## 13. Testabilidad

Evaluar si el diseño dificulta razonablemente la creación de pruebas.

Buscar:

* Dependencias excesivamente acopladas.
* Funciones con demasiadas responsabilidades.
* Efectos secundarios innecesarios.
* Lógica de negocio difícil de aislar.
* Dependencias externas imposibles de sustituir en pruebas.

**Principio:** No exigir tests para absolutamente todo código. Proporcionalidad al riesgo.

---

## 14. Mantenibilidad

Evaluar si el código puede modificarse razonablemente sin introducir errores.

Considerar:

* Complejidad.
* Acoplamiento.
* Cohesión.
* Claridad.
* Duplicación.
* Consistencia.
* Facilidad de extensión.
* Facilidad de prueba.

---

## 15. Abstracciones

Detectar tanto:

* Falta de abstracción cuando existe duplicación o complejidad real.
* Sobreabstracción cuando existen capas, clases o funciones que no aportan valor.

**Principio:** No recomendar abstracciones únicamente para "hacer el código más limpio".

---

## 16. Valores mágicos

Detectar valores literales repetidos o cuyo significado no sea evidente.

Evaluar si deberían convertirse en:

* Constantes.
* Configuración.
* Enumeraciones.
* Parámetros.

**Principio:** No convertir automáticamente todos los literales en constantes.

---

## 17. Comentarios y documentación

Detectar:

* Comentarios obsoletos.
* Comentarios que contradicen el código.
* Comentarios que explican sintaxis obvia.
* Falta de documentación en lógica realmente compleja.

**Principio:** Priorizar código claro sobre comentarios innecesarios.

---

## 18. Principio de evidencia

Cada hallazgo debe responder:

```text
¿Qué se encontró?
¿Dónde se encontró? (file_path:line_number)
¿Por qué representa un problema?
¿Qué impacto tiene?
¿Qué alternativa existe?
```

Si no existe evidencia suficiente, clasificarlo como `[INVESTIGATE]`, `[INFO]` o no reportarlo.

---

## 19. Principio de contexto

Las reglas anteriores son criterios de análisis, no reglas absolutas.

Antes de reportar un problema considerar:

```text
Contexto del proyecto
        ↓
Intención del código
        ↓
Impacto real
        ↓
Alternativas
        ↓
Recomendación
```

---

## 20. Read-only

Este skill **no modifica, crea ni elimina** archivos del proyecto. Únicamente genera la auditoría en `ai/audit/` usando `templates/AUDIT.md`.

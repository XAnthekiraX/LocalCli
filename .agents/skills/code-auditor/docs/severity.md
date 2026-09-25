# Severity Guidelines

La severidad representa el **impacto técnico** de un hallazgo, no su tamaño, complejidad ni la cantidad de código involucrado.

## CRITICAL

Problema con consecuencias potencialmente graves e inmediatas.

**Ejemplos:**

* Vulnerabilidad crítica explotable.
* Pérdida o corrupción de datos.
* Acceso no autorizado a información crítica.
* Fallo que puede inutilizar una parte fundamental del sistema.
* Operación crítica que puede producir consecuencias irreversibles.

**Acción:** Debe corregirse antes de continuar con cambios secundarios.

---

## HIGH

Problema importante con impacto significativo.

**Ejemplos:**

* Vulnerabilidad relevante.
* Error que afecta una funcionalidad importante.
* Problemas de autorización.
* Errores de consistencia de datos.
* Fallos que pueden afectar a muchos usuarios.
* Problemas de rendimiento con impacto significativo.

**Acción:** Debe recibir atención prioritaria.

---

## MEDIUM

Problema relevante pero sin consecuencias críticas.

**Ejemplos:**

* Duplicación significativa.
* Complejidad innecesaria.
* Problemas de arquitectura.
* Manejo incorrecto de errores.
* Degradaciones de rendimiento moderadas.
* Problemas que dificultan considerablemente el mantenimiento.

**Acción:** Debe planificarse su corrección.

---

## LOW

Problema de impacto reducido.

**Ejemplos:**

* Código innecesario.
* Inconsistencias menores.
* Mejoras de legibilidad.
* Pequeñas duplicaciones.
* Refactorizaciones con beneficio limitado.

**Acción:** Puede corregirse como parte del mantenimiento normal.

---

## INFO

Observación técnica que no constituye necesariamente un problema.

**Ejemplos:**

* Posible mejora futura.
* Decisión arquitectónica que merece documentación.
* Oportunidad de optimización sin impacto demostrado.
* Recomendación preventiva.

**Acción:** No requiere corrección inmediata.

---

## INVESTIGATE

Situación con **evidencia insuficiente** para confirmar o descartar un problema. No se asume severidad alta sin fundamento.

**Ejemplos:**

* Uso potencialmente dinámico no verificable.
* Patrones ambiguos que requieren validación de casos de uso.
* Comportamiento dependiente de runtime/configuración no evidente.

**Acción:** Requiere verificación adicional antes de clasificar con severidad definitiva.

---

## Criterios de evaluación

Para determinar la severidad considerar:

```text
Impacto (técnico)
   +
Probabilidad / posibilidad de explotación/ocurrencia
   +
Alcance afectado
   +
Riesgo
   +
Facilidad/dificultad de detección
```

No asignar severidad únicamente por la categoría del problema.

---

## Reglas

### No exagerar

Un problema de estilo no debe clasificarse como `HIGH` o `CRITICAL`.

### No minimizar

Un problema de seguridad o integridad de datos no debe clasificarse como `LOW` solo porque afecte pocas líneas.

### No confundir prioridad con severidad

La **severidad** describe el impacto técnico del problema.

La **prioridad** indica cuándo conviene abordarlo.

Un problema `MEDIUM` puede tener prioridad alta si bloquea una funcionalidad que se está desarrollando.

### Evidencia insuficiente

Cuando no exista evidencia suficiente para determinar el impacto de forma objetiva:

```text
INVESTIGATE
```

o, si es meramente observacional:

```text
INFO
```

**Nunca** asignar una severidad alta basándose únicamente en una suposición.

### Fuente única de verdad

Esta escala (`docs/severity.md`) es la referencia obligatoria para todos los hallazgos generados por `code-auditor`. No definir escalas alternativas dentro de reglas o plantillas.

---

## Regla principal

La severidad debe responder:

> ¿Qué tan grave es este problema para este proyecto, considerando su contexto real y evidencia concreta?

No:

> ¿Qué tan mala sería esta práctica en teoría?

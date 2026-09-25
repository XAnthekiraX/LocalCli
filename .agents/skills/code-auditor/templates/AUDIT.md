# Code Audit

> **Read-only audit.** This is a technical code audit performed in read-only mode. No source files in the project were created, modified, or deleted as part of this report. All findings are documented observations with supporting evidence.

## 1. Resumen

### Proyecto

* **Nombre:** __________
* **Stack detectado:** __________
* **Alcance:** __________
* **Fecha:** __________
* **Ruta de salida:** `ai/audit/`

### Estado general

Breve resumen técnico, objetivo y conciso de los principales hallazgos, riesgos y áreas a atender.

### Resumen de hallazgos

| Severidad | Cantidad |
|-----------|---------:|
| CRITICAL  |        0 |
| HIGH      |        0 |
| MEDIUM    |        0 |
| LOW       |        0 |
| INFO      |        0 |
| INVESTIGATE |      0 |

---

## 2. Hallazgos críticos

> Incluir únicamente problemas con severidad `CRITICAL`. Deben corregirse antes de continuar con cambios secundarios.

### [CRITICAL] Título

**ID:** AUD-001

**Ubicación:**
`file_path:line_number`

**Problema:**
Descripción concreta, objetiva y respaldada por evidencia.

**Impacto:**
Consecuencias técnicas reales (seguridad, integridad de datos, disponibilidad, etc.).

**Evidencia:**
Fragmento breve o referencia concreta que justifica el hallazgo.

**Recomendación:**
Acción concreta, realista, proporcional y accionable.

**Justificación:**
Explicación técnica de por qué mitiga o resuelve el problema.

**Categoría:** `Security` | `Correctness` | `Architecture` | `Performance` | `Maintainability` | `Style`

---

## 3. Hallazgos importantes

> Incluir problemas `HIGH` y `MEDIUM`.

### [HIGH] Título

**ID:** AUD-002

**Ubicación:**
`file_path:line_number`

**Problema:**
Descripción concreta y respaldada por evidencia.

**Impacto:**
Consecuencias técnicas.

**Evidencia:**
Referencia o snippet breve.

**Recomendación:**
Cambio concreto y accionable.

**Justificación:**
Por qué mejora el proyecto.

**Categoría:** (ver categorías arriba)

---

### [MEDIUM] Título

**ID:** AUD-003

**Ubicación:**
`file_path:line_number`

**Problema:**
Descripción concreta y respaldada por evidencia.

**Impacto:**
Consecuencias técnicas.

**Evidencia:**
Referencia o snippet breve.

**Recomendación:**
Cambio concreto y accionable.

**Justificación:**
Por qué mejora el proyecto.

**Categoría:** (ver categorías arriba)

---

## 4. Mejoras menores

> Incluir problemas `LOW`.

### [LOW] Título

**ID:** AUD-004

**Ubicación:**
`file_path:line_number`

**Problema:**
Descripción objetiva.

**Impacto:**
Impacto técnico reducido (si aplica).

**Evidencia:**
Referencia breve (si aplica).

**Recomendación:**
Mejora concreta y proporcional.

**Justificación:**
Beneficio técnico esperado.

**Categoría:** (ver categorías arriba)

---

## 5. Observaciones

> Incluir `INFO`, observaciones arquitectónicas, oportunidades de mejora futura o decisiones técnicas que merecen documentación (sin constituir necesariamente un problema).

### [INFO] Título

**ID:** AUD-005

**Ubicación:**
`file_path:line_number` (opcional)

**Observación:**
Descripción clara y objetiva.

**Recomendación:**
Opcional. Solo si aporta valor práctico.

**Justificación:**
Opcional.

**Categoría:** (ver categorías arriba)

---

## 6. Problemas que requieren investigación

> Situaciones donde no existe evidencia suficiente para confirmar o descartar un problema. No asignar severidad alta sin fundamento.

### [INVESTIGATE] Título

**ID:** AUD-006

**Ubicación:**
`file_path:line_number` (si aplica)

**Motivo:**
Qué se observó y por qué genera incertidumbre.

**Qué verificar:**
Información adicional necesaria para confirmar o descartar el problema (casos de uso, uso dinámico, flujos, tests, etc.).

**Recomendación:**
Pasos concretos de verificación (no asunciones).

**Categoría:** `Investigation`

---

## 7. Código muerto

Registrar código potencialmente no utilizado, únicamente cuando exista evidencia razonable.

| Ubicación | Elemento | Evidencia (sin referencias encontradas) | Acción sugerida | Confianza |
|---|---|---|---|---|
| `file_path` | `símbolo/función` | Búsqueda de referencias / uso detectado | Revisar / confirmar / eliminar (solo si confirmado) | Baja/Media/Alta |

**Nota:** No marcar como código muerto cuando pueda existir uso dinámico, reflection, metaprogramación o imports condicionales no comprobables.

---

## 8. Duplicación

Registrar duplicaciones relevantes (con impacto en mantenibilidad). No forzar abstracción de duplicaciones triviales.

| Ubicación(s) | Lógica duplicada | Impacto (Bajo/Medio/Alto) | Recomendación | Alcance afectado |
|---|---|---|---|---|
| `file_path1`, `file_path2` | Descripción concisa | __________ | Acción concreta y proporcional | __________ |

---

## 9. Arquitectura

### Observaciones

Describir problemas relacionados con:

* **Responsabilidades:** Claridad y cohesión.
* **Dependencias:** Dirección, acoplamiento y límites.
* **Acoplamiento/Cohesión:** Nivel y justificación de cambio.
* **Separación de capas:** Violaciones o erosión de límites.
* **Flujo de datos:** Claridad, consistencia y puntos críticos.

### Recomendaciones

Enumerar mejoras arquitectónicas relevantes, justificadas por necesidad real del proyecto.

---

## 10. Seguridad

Registrar únicamente problemas respaldados por evidencia concreta.

| ID | Problema | Ubicación (`file_path:line_number`) | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-XXX | __________ | __________ | __________ | __________ | __________ | `Security` |

---

## 11. Rendimiento

Registrar únicamente problemas con impacto razonablemente justificable (evitar optimización prematura).

| ID | Problema | Ubicación (`file_path:line_number`) | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-XXX | __________ | __________ | __________ | __________ | __________ | `Performance` |

---

## 12. Mantenibilidad

Registrar problemas relacionados con:

* **Complejidad:** Legibilidad y comprensión.
* **Consistencia:** Convenciones y patrones.
* **Testabilidad:** Aislamiento y acoplamiento.
* **Legibilidad:** Claridad semántica.
* **Acoplamiento/Cohesión:** Dificultad de cambio.
* **Duplicación:** Impacto en evolución.

| ID | Problema | Ubicación (`file_path:line_number`) | Impacto | Evidencia | Recomendación | Categoría |
|---|---|---|---|---|---|---|
| AUD-XXX | __________ | __________ | __________ | __________ | __________ | `Maintainability` |

---

## 13. Recomendaciones priorizadas

Ordenadas exclusivamente por prioridad técnica (severidad + impacto). No incluir recomendaciones ya resueltas.

1. `[CRITICAL]` AUD-XXX — Título breve
2. `[HIGH]` AUD-XXX — Título breve
3. `[MEDIUM]` AUD-XXX — Título breve
4. `[LOW]` AUD-XXX — Título breve
5. `[INFO/INVESTIGATE]` AUD-XXX — Título breve (solo si requiere acción)

---

## 14. Conclusión técnica

Resumen breve, objetivo y accionable de:

* **Problemas principales:** Riesgos más relevantes con base en evidencia.
* **Riesgos relevantes:** Mitigaciones sugeridas (proporcionales).
* **Áreas que requieren atención:** Enfoque ordenado por prioridad.
* **Mejoras recomendadas:** Quick wins vs. mejoras estructurales.

**Nota:** Esta conclusión se basa únicamente en los hallazgos documentados anteriormente. No se introducen nuevos hallazgos aquí.

# Audit Process

## 1. Descubrimiento

Antes de analizar código:

1. Identificar estructura del proyecto.
2. Detectar lenguajes (por extensiones, configuración y dependencias).
3. Detectar frameworks.
4. Detectar gestor de paquetes.
5. Revisar configuración principal.
6. Identificar arquitectura existente.
7. Identificar tests y herramientas de análisis.
8. Identificar documentación relevante.

**Regla:** No asumir tecnologías sin evidencia. El descubrimiento debe basarse en archivos reales del proyecto.

---

## 2. Definir alcance y precedencia

Determinar si la auditoría será:

* Proyecto completo.
* Backend.
* Frontend.
* Base de datos.
* Módulo específico.
* Funcionalidad específica.
* Seguridad.
* Rendimiento.
* Arquitectura.

### Precedencia entre skills

- **Frontend mayoritario (UI/componentes/páginas):** Preferir `frontend-auditor`.
- **Transversal (calidad, seguridad, arquitectura, duplicación, código muerto, multi-lenguaje):** Preferir `code-auditor`.
- **Pulido visual/motion/microinteracciones:** Preferir `impeccable`.

Si el usuario define un alcance, no analizar áreas irrelevantes. Aplicar la precedencia anterior para evitar solapamiento.

---

## 3. Analizar arquitectura

Antes de revisar detalles del código:

```text
Proyecto
  ↓
Módulos
  ↓
Dependencias
  ↓
Flujo de datos
  ↓
Responsabilidades
```

Identificar cómo está organizado realmente el proyecto.

No evaluar la arquitectura comparándola automáticamente con una arquitectura ideal. Respetar el contexto existente.

---

## 4. Aplicar reglas

Aplicar:

```text
rules/general.md
        +
reglas del lenguaje (detectadas por evidencia)
        +
reglas del framework (detectadas por evidencia)
        +
reglas relevantes del proyecto
```

Las reglas son criterios de análisis, no requisitos absolutos. `general.md` se aplica **siempre**. Solo se incluyen reglas específicas cuando existen archivos o dependencias que lo justifiquen.

---

## 5. Recopilar evidencia

Para cada posible problema:

```text
Encontrar
   ↓
Verificar (existencia real)
   ↓
Comprender contexto
   ↓
Evaluar impacto
```

No reportar una sospecha como un problema confirmado. Cada hallazgo debe estar respaldado por evidencia concreta (`file_path:line_number` cuando sea posible).

---

## 6. Evaluar impacto

Determinar:

* Qué puede romperse.
* A quién afecta.
* Con qué frecuencia puede ocurrir.
* Qué tan difícil sería detectarlo.
* Qué tan difícil sería corregirlo.
* Si existe riesgo de seguridad, pérdida de datos o degradación del sistema.
* Alcance afectado.

---

## 7. Clasificar

Asignar severidad siguiendo estrictamente `docs/severity.md`:

```text
CRITICAL
HIGH
MEDIUM
LOW
INFO
INVESTIGATE
```

La severidad debe representar el impacto técnico real y no la cantidad de código involucrado. Si no hay evidencia suficiente para determinar impacto, usar `INVESTIGATE` o `INFO`.

---

## 8. Eliminar falsos positivos

Antes de incluir un hallazgo:

```text
¿Existe evidencia concreta?
        ↓
¿Entiendo el contexto completo?
        ↓
¿Es realmente un problema (no una elección intencional)?
        ↓
¿Tiene impacto demostrable?
        ↓
¿La recomendación aporta valor práctico?
        ↓
¿Es proporcional al problema?
```

Si alguna respuesta es negativa, reconsiderar el hallazgo. No forzar la inclusión de hallazgos superficiales.

---

## 9. Buscar problemas relacionados

Cuando se encuentre un problema importante, revisar si existe el mismo patrón en otras partes del proyecto.

Ejemplo:

```text
Problema encontrado
        ↓
Buscar mismo patrón (con evidencia)
        ↓
Agrupar hallazgos relacionados por causa raíz
```

Evitar generar decenas de hallazgos idénticos cuando representan una misma causa raíz. Preferir agrupar y describir alcance afectado.

---

## 10. Identificar causa raíz

Cuando sea posible, distinguir entre:

```text
Síntoma
   ↓
Problema
   ↓
Causa raíz
```

Priorizar la causa raíz sobre múltiples síntomas derivados. Formular recomendaciones orientadas a corregir la causa raíz cuando sea realista.

---

## 11. Formular recomendaciones

Cada recomendación debe ser:

* Concreta.
* Realista.
* Compatible con el proyecto (respetar contexto existente).
* Proporcional al problema.
* Accionable (indicar qué cambiar, no solo qué evitar).

Evitar recomendaciones genéricas como:

```text
"Mejorar el código."
"Aplicar buenas prácticas."
"Refactorizar."
```

Explicar qué debería cambiar y por qué, con base en evidencia.

---

## 12. Revisar auditorías anteriores

Si existe una auditoría anterior en `ai/audit/`:

```text
Auditoría anterior (ai/audit/)
        ↓
Comparar hallazgos (ID cuando existan)
        ↓
Verificar estado actual
        ↓
Mantener / actualizar / cerrar
```

No repetir automáticamente problemas ya solucionados. Actualizar hallazgos existentes en lugar de duplicarlos cuando sea posible.

---

## 13. Generar resultado

Utilizar **obligatoriamente**:

```text
templates/AUDIT.md
```

El resultado debe escribirse en:

```text
ai/audit/
```

El informe debe permitir que otro desarrollador pueda comenzar a corregir los problemas sin repetir toda la investigación. Incluir IDs de hallazgos, ubicación (`file_path:line_number`), evidencia, impacto, recomendación y justificación.

**Modo de operación:** Read-only. No se modifican archivos del proyecto durante la auditoría.

---

## 14. Regla final

Una auditoría de calidad debe producir:

```text
Pocos hallazgos relevantes + bien fundamentados
        >
Muchos hallazgos superficiales
```

La precisión, objetividad y utilidad de los hallazgos tienen prioridad absoluta sobre la cantidad.

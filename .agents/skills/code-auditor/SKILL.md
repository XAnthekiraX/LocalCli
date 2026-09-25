---
name: code-auditor
description: Use this skill when auditing code quality, security, architecture, performance, maintainability, or best practices across a codebase (multi-language or full-stack). Use ONLY when the task is a technical code audit (not pure UX/visual design). Prefer frontend-auditor for frontend-focused UI/UX analysis, and impeccable for visual polish/motion. Detects project stack automatically, applies general + technology-specific rules, produces a read-only prioritized audit report with concrete recommendations.
---

# Code Auditor

## Objetivo

Realizar auditorías técnicas sobre proyectos existentes para identificar problemas de calidad, mantenibilidad, arquitectura, seguridad, rendimiento y buenas prácticas.

La skill analiza el proyecto antes de emitir conclusiones y adapta la auditoría al stack detectado.

**La skill NO modifica código. Es estrictamente read-only.**

Su función es:

```text
Analizar → Detectar → Evaluar → Documentar → Recomendar
```

### Alcance y precedencia

- **Transversal (este skill):** Calidad de código, seguridad, arquitectura, duplicación, código muerto, complejidad y mantenibilidad (multi-lenguaje/full-stack).
- **Frontend-específico (frontend-auditor):** Arquitectura/rendimiento/accesibilidad/UX estructurado centrado en UI.
- **Visual/Polish (impeccable):** Jerarquía visual, tipografía, espaciado, color, microinteracciones y pulido de UI.

**Regla de precedencia:** Si el alcance es mayoritariamente frontend (UI/componentes/páginas/vistas), priorizar `frontend-auditor`. Si es transversal (calidad de código, seguridad, arquitectura, patrones, multi-lenguaje), utilizar `code-auditor`. Evitar duplicidad entre skills.

---

## Flujo de auditoría

### 1. Descubrimiento

Analizar primero la estructura general del proyecto.

Identificar:

* Lenguajes utilizados
* Frameworks
* Librerías principales
* Sistema de build
* Gestor de paquetes
* Arquitectura
* Estructura de directorios
* Configuración relevante
* Tests
* Scripts disponibles
* Base de datos, si existe

No asumir el stack sin comprobarlo.

---

### 2. Selección de reglas

Aplicar siempre:

```text
rules/general.md
```

Después aplicar las reglas correspondientes al stack detectado.

Ejemplo:

```text
TypeScript + React + Node.js
        ↓
general.md
typescript.md
react.md
node.md
```

No aplicar reglas de tecnologías que no formen parte del proyecto. La selección se basa en evidencia (archivos, extensiones, configuración y dependencias detectadas).

---

### 3. Auditoría

Analizar, cuando corresponda:

#### Código

* Código muerto
* Código duplicado
* Funciones innecesariamente complejas
* Responsabilidades mezcladas
* Nombres poco claros
* Abstracciones innecesarias
* Código difícil de mantener
* Patrones inconsistentes
* Lógica repetida
* Manejo incorrecto de errores
* Validaciones insuficientes

#### Arquitectura

* Separación de responsabilidades
* Dependencias entre capas
* Acoplamiento
* Cohesión
* Organización de módulos
* Violaciones de arquitectura
* Duplicación de lógica entre capas
* Límites incorrectos entre componentes

#### Seguridad

* Validación de entradas
* Manejo de datos sensibles
* Autenticación/autorización
* Exposición de información
* Configuración insegura
* Dependencias vulnerables cuando pueda verificarse
* Problemas comunes específicos del stack

#### Rendimiento

* Operaciones innecesarias
* Consultas ineficientes
* Procesamiento repetido
* Problemas de renderizado
* Cálculos innecesarios
* Uso inadecuado de memoria
* Problemas específicos del stack

#### Mantenibilidad

* Consistencia
* Legibilidad
* Complejidad
* Testabilidad
* Facilidad para extender funcionalidades
* Dependencias innecesarias
* Configuración excesivamente compleja

---

## 4. Evidencia

Cada hallazgo debe estar respaldado por evidencia concreta del proyecto.

Cuando sea posible indicar:

```text
Archivo
Línea o función
Problema
Por qué es un problema
Impacto
```

No reportar problemas basándose únicamente en suposiciones.

Distinguir entre:

```text
Hecho observado
Inferencia técnica
Recomendación
```

---

## 5. Clasificación

Cada hallazgo debe tener una severidad, siguiendo `docs/severity.md` como fuente única de verdad:

```text
CRITICAL
HIGH
MEDIUM
LOW
INFO
INVESTIGATE
```

La justificación de la severidad debe basarse en impacto técnico real, no en cantidad de líneas.

---

## 6. Priorización

Ordenar los hallazgos por:

1. Impacto
2. Riesgo
3. Severidad
4. Facilidad de corrección

No priorizar únicamente por cantidad de problemas encontrados.

---

## 7. Auditoría existente

Si existe una auditoría previa en:

```text
ai/audit/
```

revisarla antes de comenzar.

Determinar:

* Problemas ya solucionados
* Problemas todavía presentes
* Nuevos problemas
* Hallazgos que ya no son relevantes

No duplicar automáticamente hallazgos existentes. Comparar, actualizar o cerrar hallazgos según corresponda.

---

## 8. Resultado

Generar la auditoría utilizando obligatoriamente:

```text
templates/AUDIT.md
```

La salida debe escribirse en:

```text
ai/audit/
```

Usar nombres claros y consistentes (p.ej. `AUDIT.md` para el informe principal del alcance solicitado). Si se realiza un re-audit, conservar histórico razonable sin duplicar contenido innecesariamente.

Cada hallazgo debe seguir la estructura estandarizada del template:

```text
### [SEVERITY] Título

**ID:** AUD-XXX

**Ubicación:**
file_path:line_number

**Problema:**
Descripción concreta y objetiva.

**Impacto:**
Qué consecuencias técnicas tiene.

**Evidencia:**
Snippet breve o referencia concreta.

**Recomendación:**
Acción concreta, realista y proporcional.

**Justificación:**
Por qué resuelve o mitiga el problema.
```

---

## Reglas importantes

### Solo lectura (Read-only)

Este skill **no crea, elimina ni modifica** archivos del proyecto. Únicamente genera documentación en `ai/audit/`.

### No sobreingenierizar

No recomendar patrones, abstracciones o arquitecturas únicamente porque sean consideradas "best practices". La recomendación debe estar justificada por una necesidad real del proyecto.

### No buscar problemas artificialmente

Si una parte del proyecto está correctamente implementada, no generar un hallazgo simplemente para completar la auditoría.

### Priorizar problemas reales

Preferir:

```text
Problema real + evidencia + impacto + solución
```

sobre:

```text
Preferencia personal
```

### Respetar el contexto del proyecto

Una práctica puede ser apropiada en un proyecto y no serlo en otro.

Considerar:

* Tamaño
* Complejidad
* Objetivo
* Arquitectura existente
* Stack
* Etapa del proyecto

---

## Alcance

La auditoría puede abarcar todo el proyecto o una parte específica.

Si el usuario especifica un alcance (ej.:

```text
audita backend
audita frontend
audita autenticación
audita API
audita inventario
```

), limitar el análisis a ese contexto, incluyendo únicamente las dependencias necesarias para comprenderlo.

---

## Principio principal

La auditoría debe responder:

```text
¿Qué está mal?
¿Por qué está mal?
¿Dónde está?
¿Qué impacto tiene?
¿Cómo debería mejorarse?
¿Qué prioridad tiene?
```

No debe convertirse en una refactorización automática ni en una reescritura del proyecto.

---

## Principio de evidencia

Cada hallazgo requiere evidencia suficiente. Si no existe evidencia para confirmar el impacto, clasificar como `[INVESTIGATE]` o `[INFO]`, nunca asumir severidad alta sin fundamento.

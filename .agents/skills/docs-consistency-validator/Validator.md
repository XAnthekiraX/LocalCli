# Validator — Validación de la documentación

Procedimiento detallado de la **Fase VALIDACIÓN** de la skill
`docs-consistency-validator`. Es **solo lectura**: nunca modifica, crea ni
borra archivos. Referencia la estructura canónica definida en `SKILL.md`
(no se duplica aquí).

# Flujo de validación

## 1. Ubicar la documentación

Determinar el directorio de docs del proyecto. Orden de precedencia:

1. Variable de anulación explícita si se provee (p.ej. `docs_dir=/ruta/a/docs`).
2. El directorio de trabajo del proyecto.

## 2. Detectar la estructura actual

Clasificar la estructura encontrada en `ai/docs/`:

- **BY-LAYER:** hay `backend/`, `frontend/`, `database/` como capas (con o
  sin carpetas numeradas internas).
- **FLAT/LEGACY:** documentos sueltos directamente en `ai/docs/` sin capas.
- **NOMBRES VIEJOS:** capas presentes pero con entry points o carpetas
  antiguas (`00-BACKEND.md`, `00-FRONTEND.md`, `DATABASE.md`, `schema/`,
  `rules/`, `operations/`, `user-flow/`, `components/`, documentos en raíz).

Reportar qué estructura se detectó antes de validar.

## 3. Construir el mapa actual por capa

Para cada capa, listar qué existe realmente:

```
ai/docs/<capa>/
  <archivos en raíz>
  <carpetas> / <archivos dentro>
```

Separar explícitamente lo que está en la raíz de lo que está dentro de
subcarpetas.

## 4. Comparar contra la estructura canónica

La estructura canónica está en `SKILL.md` (sección "Estructura canónica").
Comparar uno a uno:

- **Raíz:** ¿están los archivos de raíz esperados (`ARCHITECTURE.md` y los
  que la capa defina en raíz)?
- **Carpetas:** ¿están las carpetas con su prefijo numérico correcto
  (`01-schema/`, `02-…`, etc.)?
- **Documentos:** ¿cada documento está en su subcarpeta esperada?
- **Extra:** ¿hay archivos/carpetas que no deberían existir?

Para cada desviación, anotar: archivo/carpeta actual → destino esperado
según la estructura canónica.

## 5. Categorizar hallazgos estructurales

Clasificar cada desviación usando la tabla de severidad de `STANDARD.md`.
Criterios de referencia (ajustar al rango correcto según el caso):

| # | Hallazgo | Severidad sugerida |
|---|----------|--------------------|
| 1 | Falta `ARCHITECTURE.md` en la raíz de una capa | ERROR |
| 2 | Entry point viejo presente (`00-BACKEND.md`, `00-FRONTEND.md`, `DATABASE.md`) | WARNING (migrable) |
| 3 | Carpeta sin prefijo numérico (`user-flow/`, `schema/`, …) | WARNING |
| 4 | Documento en raíz que debería estar en subcarpeta (`STATE.md` en raíz) | WARNING |
| 5 | Estructura flat/legacy (sin capas) | WARNING (recomendación migrar a by-layer) |
| 6 | Carpeta/archivo huérfano que no corresponde a la estructura canónica | INFO |
| 7 | Arquitectura canónica desactualizada vs realidad del proyecto | ERROR (consistencia) |

Elegir siempre la severidad más alta que aplique a cada hallazgo.

## 6. Validación de cumplimiento de las reglas del subagente de documentación

Verificar que toda la documentación cumple las 5 reglas de
`~/.config/opencode/agents/documentation_agent.md`:

1. **Ubicación:** toda la documentación vive dentro de `ai/docs/`. Un
   documento fuera de `ai/docs/` → hallazgo.
2. **Referencia con wiki link:** un documento que depende de información en
   otro archivo debe referenciarlo con formato wiki link de Obsidian:
   `[[nombre-archivo]]` (misma capa) o `[[ruta/completa/Archivo]]` (otra capa).
   Dependencia sin referencia, o con formato incorrecto → hallazgo.
3. **Solo contexto necesario:** las referencias apuntan únicamente al
   contexto necesario, no a información no relacionada del archivo destino.
   Referencia sobreamplia → hallazgo (WARNING).
4. **Formato obligatorio:** la referencia usa el formato wiki link
   `[[nombre-archivo]]` o `[[ruta/completa/Archivo]]`. Cualquier otro
   formato → hallazgo.
5. **Referencias vigentes:** si una modificación (renombre, movimiento,
   reescritura) cambia las líneas que una referencia existente usa, esa
   referencia debe haberse actualizado. En este modo VALIDACIÓN, comprobar
   que las referencias apunten a líneas que realmente existen y corresponden
   al contenido referenciado; una referencia rota o desactualizada →
   hallazgo. (La actualización en sí la ejecuta el modo MIGRACIÓN.)
6. **Idioma (español):** el **prosa narrativo** de cada documento (secciones,
   descripciones, guías) debe estar en español. El código, identificadores,
   nombres técnicos y JSON no cuentan. Un documento cuyo prosa esté en otro
   idioma → hallazgo. El contenido que vaya a centralizarse en
   `ARCHITECTURE.md` desde un entry point en otro idioma también se marca.

Para el hallazgo de idioma, `fix_sugerido` = "traducir a español". La
traducción NO la ejecuta esta skill; se delega a otro proceso (se marca como
"requiere traducción").

Un documento que depende de otro referencia su fuente con ruta y líneas; no
duplica el contenido.

## 7. Validación de coherencia de contenido

Además de la estructura, validar el contenido entre capas (tipos,
referencias, terminología). Estas reglas de coherencia se detallan en el
estándar de coherencia (`STANDARD.md`, secciones de tipo/terminología):

- Mismo atributo con tipo distinto en storage / DTO / UI → hallazgo.
- Renombrado de la misma entidad/atributo entre documentos → hallazgo.
- Endpoint referenciado por el frontend sin existir en backend (y viceversa)
  → hallazgo.
- **Cohesión por archivo:** verificar que cada archivo cumpla "un archivo =
  una responsabilidad" (regla 2 de project-planner). Un archivo que mezcle
  dominios no relacionados (p.ej. seguridad + testing en el mismo archivo)
  → hallazgo WARNING.

## 8. Validación de optimización (nueva)

Detectar redundancias y oportunidades de mejora en la documentación.
Esta fase es **solo lectura** y genera hallazgos con `optimizacion_sugerida`.

### 8.1 Detección de contenido duplicado

Para cada concepto/definición encontrada en la documentación:

1. Identificar si el mismo concepto aparece en más de un archivo.
2. Determinar cuál es la fuente más completa (más detalle, más contexto).
3. Los demás archivos que lo duplican → hallazgo WARNING con
   `optimizacion_sugerida` = "referenciar [[archivo-fuente]] en lugar de
   duplicar".

### 8.2 Detección de información fragmentada

Para cada concepto que debería ser unitario:

1. Verificar si toda su definición está en un solo archivo.
2. Si está repartida entre 2+ archivos → hallazgo INFO con
   `optimizacion_sugerida` = "centralizar en [[archivo-mejor-ubicado]]".

### 8.3 Verificación contra ARCHITECTURE.md

Para cada capa, verificar que ARCHITECTURE.md centralice la información
global:

1. Si un archivo de subcarpeta contiene información que ya está resumida
   en ARCHITECTURE.md → hallazgo INFO con `optimizacion_sugerida` =
   "eliminar duplicación; ARCHITECTURE.md ya centraliza esta info".

### 8.4 Reglas de planificación (project-planner)

Verificar que la documentación generada cumpla las reglas de
project-planner cuando apliquen:

1. **SPEC antes de arquitectura:** si existe documentación de arquitectura
   sin SPEC correspondiente → hallazgo WARNING.
2. **Un archivo = una responsabilidad:** archivos que mezclan múltiples
   dominios → hallazgo WARNING (ya cubierto en 7.cohesión).
3. **Escritura directa:** documentos con exceso de jerga técnica o texto
   innecesariamente largo → hallazgo INFO con `optimizacion_sugerida` =
   "simplificar redacción".

## 9. Emitir el reporte

Generar el reporte estructurado según `STANDARD.md`.

- `validation_report.md` — reporte legible: estructura detectada, resumen por
  severidad, lista completa de hallazgos (incluye los estructurales), y
  sección de excepciones aceptadas.
- `validation_report.json` — mismos hallazgos como JSON parseable.

Cada hallazgo estructural incluye en `fix_sugerido` el destino exacto según
la estructura canónica (p.ej. `mover STATE.md a 05-state/STATE.md`).

## 10. Resumen en consola

Devolver al usuario un resumen breve:

- Número de hallazgos por severidad.
- Para cada `CRITICAL` y `ERROR`: archivo(s), concepto y `fix_sugerido`.
- Rutas de los archivos de reporte generados.
- Indicar si la estructura cumple la canónica o si se requiere el modo
  MIGRACIÓN para normalizarla.

---

# Reglas del validador

- **Nunca** modificar, crear ni eliminar archivos de documentación.
- No inventar entidades, campos, tipos, endpoints o rutas.
- Cuando las convenciones declaradas por el proyecto entren en conflicto con
  `STANDARD.md`, la declaración del proyecto gana; anotarlo en el reporte.
- Reportar hallazgos con un `fix_sugerido` concreto, nunca como consejo vago.
- Si tras la validación hay hallazgos estructurales migrables, indicar que se
  recomienda ejecutar el modo MIGRACIÓN (ver `Migrator.md`).

---
name: obsidian-style-guide
description: >
  Guía de estilo para documentación compatible con Obsidian. Define reglas de
  naming, referencias wiki links y estructura de archivos para todas las skills
  de documentación.
---

# Guía de Estilo para Documentación Obsidian

Esta guía define las convenciones que todas las skills de documentación deben seguir para generar documentación compatible con Obsidian.

---

## Reglas de Referencia

### Formato de Wiki Links

**SIEMPRE** usar formato wiki link con ruta relativa:

```
[[carpeta/ARCHIVO]]
```

**Ejemplos:**
- `[[database/TABLES]]` en lugar de `ai/docs/database/TABLES.md`
- `[[backend/API-AUTH]]` en lugar de `ai/docs/backend/02-api/API-AUTH.md`
- `[[frontend/FRONTEND]]` en lugar de `ai/docs/frontend/FRONTEND.md`

### Reglas Específicas

1. **NUNCA** usar rutas absolutas: `/ruta/archivo.md`
2. **NUNCA** usar rutas relativas con `../`: `../../archivo.md`
3. **SIEMPRE** incluir la carpeta principal: `[[database/TABLES]]`
4. **SIEMPRE** usar mayúsculas para nombres de archivo: `[[TABLES]]`, no `[[tables]]`
5. **USAR** pipes para alias cuando sea necesario: `[[database/TABLES|ver tablas]]`

---

## Reglas de Naming (Nombres de Archivo)

### Sufijos para Evitar Duplicados

| Capa | Sufijo | Ejemplo |
|------|--------|---------|
| Backend | `-BACK` | `API-AUTH-BACK.md` |
| Frontend | `-FRONT` | `COMPONENTS-FRONT.md` |
| Database | `-DB` | `TABLES-DB.md` |

### Convenciones de Nombre

1. **Un recurso = Un archivo:**
   - `API-AUTH.md` (agrupa todos los endpoints de auth)
   - `API-USERS.md` (agrupa todos los endpoints de users)

2. **DTOs por recurso:**
   - `dto/AUTH-DTO.md`
   - `dto/USERS-DTO.md`

3. **Archivos principales de capa:**
   - `FRONTEND.md` (antes `ARCHITECTURE.md`)
   - `BACKEND.md` (antes `ARCHITECTURE.md`)
   - `DATABASE.md` (antes `ARCHITECTURE.md`)

---

## Estructura de Carpetas

```
ai/docs/
├── PROJECT.md
├── frontend/
│   ├── FRONTEND.md
│   ├── DOMAIN.md
│   ├── 01-user-flow/
│   ├── 02-components/
│   ├── 03-data/
│   ├── 04-behavior/
│   ├── 05-state/
│   ├── 06-validation/
│   ├── 07-errors/
│   ├── 08-auth/
│   └── 09-api-dependencies/
├── backend/
│   ├── BACKEND.md
│   ├── DECISIONS.md
│   ├── 01-domain/
│   ├── 02-api/
│   │   ├── API-GENERAL.md
│   │   ├── API-AUTH.md
│   │   ├── API-USERS.md
│   │   └── dto/
│   ├── 03-security/
│   ├── 04-infrastructure/
│   └── 05-quality/
└── database/
    ├── DATABASE.md
    ├── 01-schema/
    ├── 02-rules/
    └── 03-operations/
```

---

## Reglas de Contenido

### Referencias Internas

```markdown
## Tablas relacionadas

Ver [[database/TABLES]] para la estructura completa.

## DTOs de referencia

Ver [[backend/dto/AUTH-DTO]] para los esquemas de request/response.
```

### Referencias entre Capas

```markdown
## Comunicación con Backend

El frontend consume los endpoints documentados en [[backend/API-AUTH]].

## Validación

Las reglas de negocio están en [[database/BUSINESS_RULES]].
```

---

## Checklist de Validación

Antes de publicar documentación, verificar:

- [ ] Todas las referencias usan formato `[[carpeta/ARCHIVO]]`
- [ ] No hay rutas absolutas `/ruta/archivo.md`
- [ ] No hay nombres duplicados entre capas
- [ ] Los archivos principales son `FRONTEND.md`, `BACKEND.md`, `DATABASE.md`
- [ ] Cada recurso tiene su propio archivo (`API-<RECURSO>.md`)
- [ ] Los DTOs están en `dto/<RECURSO>-DTO.md`
- [ ] Los nombres usan mayúsculas y guiones

---

## Ejemplo de Documento con Referencias

```markdown
# API de Autenticación

## Endpoints

- POST /auth/login
- POST /auth/register
- POST /auth/logout

## DTOs

Ver [[backend/dto/AUTH-DTO]] para esquemas de request/response.

## Reglas de Negocio

Ver [[database/BUSINESS_RULES]] para validaciones.

## Seguridad

Ver [[backend/SECURITY]] para políticas de autenticación.
```

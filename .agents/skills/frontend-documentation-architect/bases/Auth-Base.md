# Base de AUTH.md

Plantilla de contenido para `ai/docs/frontend/08-auth/AUTH.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- No documenta cómo el backend genera/valida JWT, sesiones ni hashes.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Flujo de Login

- Flujo de inicio de sesión del usuario.
- Referencia a [[backend/API-AUTH]] para endpoints de autenticación.

## 2. Registro

- Flujo de registro de usuario.

## 3. Logout

- Flujo de cierre de sesión.

## 4. Estado Autenticado/No Autenticado

- Estados de autenticación de la UI.

## 5. Persistencia de Sesión

- Cómo se persiste la sesión.

## 6. Protección de Rutas

- Rutas protegidas.

## 7. Redirecciones

- Redirecciones por autenticación.

## 8. Expiración de Sesión

- Comportamiento ante expiración.

## 9. Comportamiento ante 401/403

- Qué hace la UI ante 401/403.

## 10. Información del Usuario

- Información del usuario utilizada por la UI.

# Guía de entrevista

Para documentar AUTH, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Flujos de login, registro y logout definidos.
- Protección de rutas y redirecciones documentadas.
- Comportamiento ante 401/403 explicitado.
- No documenta la implementación del backend.
- Referencias a [[backend/API-AUTH]] para endpoints.

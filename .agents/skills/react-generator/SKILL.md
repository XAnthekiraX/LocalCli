---
name: react-generator
description: >
  Use this skill when generating, refactoring, or analyzing React components, hooks, and patterns.
  It provides guidelines for component structure, hooks usage, state management, and React best practices.
---
# React Generator Skill

## Objetivo de la Skill

Eres un generador de aplicaciones React especializado en transformar un prototipo HTML en una arquitectura React moderna y escalable.

Tu función es convertir un `index.html` (prototipo visual) en un proyecto React + Tailwind manteniendo la fidelidad visual y estructural, utilizando la documentación UX/UI y la especificación de API como contexto de diseño y datos.

El HTML es únicamente referencia visual, no implementación final.

---

## Entradas

La Skill recibe:

### 1. Prototipo
- index.html

### 2. Documentación UX/UI
- pages
- layouts
- sections
- components
- forms
- navigation
- user flows
- data requirements

### 3. Documentación API
- overview
- public-api
- private-api
- authentication
- data-models
- validation
- errors

---

## Salidas

Debe generar un proyecto React estructurado:

- components/
- pages/
- hooks/
- services/
- lib/
- types/
- assets/
- App.tsx
- main.tsx

---

## Responsabilidades

La Skill debe:

- Convertir HTML en componentes React reutilizables.
- Separar UI por secciones lógicas.
- Detectar páginas y rutas.
- Crear arquitectura de routing.
- Integrar datos desde APIs definidas.
- Crear hooks para consumo de datos.
- Generar types desde data-models.
- Convertir eventos del HTML a React events.
- Adaptar estilos a Tailwind CSS.
- Eliminar lógica DOM imperativa.
- Mantener fidelidad visual al prototipo.

---

## Reglas críticas de estructura (OBLIGATORIO)

### 1. Un componente por archivo

Cada archivo `.tsx` debe contener **exactamente un solo componente React principal**.

❌ Prohibido:

```tsx
// Dos o más componentes en el mismo archivo
export function Hero() {}
export function HeroButton() {}
```

✔ Correcto:

- Hero.tsx → solo Hero
- HeroButton.tsx → solo HeroButton

---

### 2. Sin subcomponentes en el mismo archivo

No se permiten componentes internos dentro del mismo archivo si representan piezas reutilizables.

❌ Prohibido:

```tsx
function Hero() {
  function Button() {}
}
```

✔ Correcto:

- Button.tsx separado

---

### 3. Excepción controlada

Solo se permite subcomponente interno si:

- es puramente visual
- no es reutilizable
- no tiene lógica propia
- no se usa en otro lugar

Aun así, se recomienda externalizarlo.

---

## Reglas de integración con API

### 1. Uso obligatorio de documentación API

Cuando exista API definida:

- debe ser usada obligatoriamente
- no se permiten endpoints inventados
- no se permiten mocks si existe API

---

### 2. Tipado obligatorio

Todos los datos deben derivarse de `data-models`.

Ejemplo:

```ts
type Project = {
  id: string
  title: string
  description: string
}
```

---

### 3. Hooks obligatorios

Cada recurso API debe tener su hook:

- useProjects
- useProject
- useAuth
- useContact

---

## Reglas de UI y estilos

- Usar Tailwind CSS exclusivamente
- No usar CSS externo del prototipo como fuente final
- Convertir intención visual, no implementación
- No usar estilos inline arbitrarios

---

## Reglas de comportamiento

- Eventos HTML deben convertirse a eventos React
- No usar DOM API (querySelector, addEventListener)
- No manipular DOM directamente

---

## Reglas de arquitectura

### Componentización

Basada en UX/UI Analyzer:

- Navbar
- Hero
- About
- Projects
- ContactForm
- Footer

---

### Separación estricta

- components/ → UI reutilizable
- pages/ → vistas completas
- hooks/ → lógica de datos
- services/ → acceso a API
- types/ → modelos

---

## Reglas de datos

### Caso 1: datos con API

```ts
const data = await api.getProjects()
```

### Caso 2: sin API

```ts
const [state, setState] = useState()
```

---

## Proceso de transformación

1. Leer index.html
2. Leer UX/UI docs
3. Leer API docs
4. Identificar layout global
5. Separar en páginas
6. Separar en componentes
7. Detectar datos dinámicos
8. Mapear datos a API
9. Crear hooks
10. Crear types
11. Convertir a React components
12. Aplicar Tailwind
13. Eliminar lógica imperativa
14. Validar estructura final

---

## Reglas de decisión de API

- Mostrar datos → GET
- Enviar formulario → POST
- Editar → PATCH/PUT
- Eliminar → DELETE

Si no existe API:

- usar estado local
- no inventar endpoints

---

## Nivel de abstracción

Trabaja a nivel de arquitectura frontend.

No interpreta implementación del HTML.

Solo interpreta intención del diseño y contratos de datos.

---

## Configuración del proyecto

Definida por el usuario:

- ruta de entrada index.html
- rutas de documentación UX/UI
- rutas de documentación API
- framework base
- convenciones de naming
- routing strategy
- estructura de carpetas
- idioma

Esta configuración tiene prioridad sobre todas las reglas genéricas.

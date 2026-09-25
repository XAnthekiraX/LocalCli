# React Audit Rules

Reglas específicas para proyectos React.

## 1. Componentes

Revisar:

* Componentes excesivamente grandes.
* Componentes con demasiadas responsabilidades.
* Lógica de negocio mezclada con presentación.
* Duplicación entre componentes.
* Props innecesariamente numerosas.
* Componentes difíciles de reutilizar o probar.

No dividir componentes únicamente por cantidad de líneas.

---

## 2. Estado

Detectar:

* Estado duplicado.
* Estado derivado almacenado innecesariamente.
* Estado colocado en un nivel incorrecto.
* Estado global utilizado cuando el problema puede resolverse localmente.
* Estado local utilizado cuando realmente debe compartirse.

Evaluar primero el flujo real de datos.

---

## 3. Props

Revisar:

* Prop drilling excesivo.
* Props que nunca se utilizan.
* Props redundantes.
* Props que representan estado derivado.
* Interfaces de props excesivamente complejas.

No considerar prop drilling como problema automáticamente; evaluar su profundidad y complejidad.

---

## 4. Hooks

Revisar:

* Dependencias incorrectas en `useEffect`.
* `useEffect` utilizado para lógica que podría ejecutarse directamente durante el render.
* Efectos innecesarios.
* Hooks personalizados duplicados.
* Hooks utilizados fuera de sus reglas.
* Estado o efectos que pueden simplificarse.

Especial atención a:

```text id="6x4s4b"
render
  ↓
useEffect
  ↓
setState
  ↓
render nuevamente
```

cuando ese ciclo no sea necesario.

---

## 5. `useEffect`

Detectar:

* Dependencias faltantes.
* Dependencias innecesarias.
* Efectos que realizan múltiples responsabilidades.
* Limpieza (`cleanup`) ausente cuando es necesaria.
* Fetches o suscripciones sin cancelación cuando exista riesgo de race conditions.
* Efectos utilizados para sincronizar valores que ya pueden derivarse directamente.

No recomendar eliminar un `useEffect` sin analizar su propósito.

---

## 6. Renderizado

Buscar:

* Renderizados innecesarios.
* Cálculos costosos ejecutados en cada render.
* Creación innecesaria de objetos o funciones.
* Listas grandes sin estrategia adecuada.
* Componentes que se actualizan por cambios de estado que no necesitan.

No utilizar `memo`, `useMemo` o `useCallback` automáticamente.

---

## 7. Memoización

Evaluar:

* `React.memo`.
* `useMemo`.
* `useCallback`.

Detectar:

* Memoización innecesaria.
* Memoización aplicada sin beneficio real.
* Dependencias incorrectas.
* Complejidad introducida únicamente para evitar renders insignificantes.

La optimización debe estar justificada por un problema real.

---

## 8. Listas

Revisar:

* Uso correcto de `key`.
* Keys inestables.
* Uso innecesario del índice como `key`.
* Listas excesivamente grandes.
* Componentes de lista demasiado complejos.

No considerar el índice incorrecto en todos los casos; evaluar si la lista puede cambiar de orden, insertar o eliminar elementos.

---

## 9. Formularios

Detectar:

* Estado duplicado del formulario.
* Validaciones inconsistentes.
* Inputs controlados y no controlados mezclados incorrectamente.
* Lógica de formulario excesivamente compleja.
* Manejo incorrecto de errores o estados de carga.

---

## 10. Datos y API

Revisar:

* Fetches duplicados.
* Peticiones realizadas innecesariamente.
* Estado de carga inexistente cuando es necesario.
* Errores de API ignorados.
* Datos transformados repetidamente.
* Lógica de acceso a API mezclada excesivamente con componentes.

Evaluar si la lógica debería estar en hooks, servicios o una capa de datos según la arquitectura existente.

---

## 11. Context

Detectar:

* Context usado como almacenamiento global indiscriminado.
* Contextos excesivamente grandes.
* Actualizaciones que provocan renders innecesarios.
* Datos que podrían permanecer como estado local.

No recomendar reemplazar Context automáticamente por otra solución.

---

## 12. Estado global

Si existe Redux, Zustand, Context u otra solución:

Revisar:

* Estado innecesariamente global.
* Selectores ineficientes.
* Mutaciones incorrectas.
* Duplicación entre estado global y local.
* Stores excesivamente grandes.
* Lógica de negocio dispersa.

Aplicar las reglas específicas de la librería cuando exista un archivo correspondiente.

---

## 13. Arquitectura

Evaluar separación entre:

```text id="f2f2e8"
Componentes
    ↓
Hooks
    ↓
Lógica de aplicación
    ↓
Servicios / API
```

Detectar cuando un componente concentra demasiadas capas de responsabilidad.

No imponer esta estructura si el proyecto utiliza otra arquitectura coherente.

---

## 14. Accesibilidad

Revisar cuando sea relevante:

* Elementos interactivos sin semántica adecuada.
* Inputs sin labels.
* Imágenes sin `alt` cuando corresponde.
* Navegación mediante teclado.
* Uso incorrecto de elementos HTML.
* Información comunicada únicamente mediante color.

---

## 15. Seguridad

Buscar:

* Uso inseguro de `dangerouslySetInnerHTML`.
* Datos externos insertados directamente en HTML.
* Información sensible expuesta en el cliente.
* Tokens almacenados de forma insegura cuando sea relevante.
* Validaciones asumidas únicamente desde el frontend.

Recordar que las validaciones del frontend no sustituyen las del backend.

---

## 16. Performance

Analizar cuando exista evidencia de impacto:

* Renderizados innecesarios.
* Listas grandes.
* Imágenes excesivamente pesadas.
* Bundles innecesariamente grandes.
* Imports que aumentan significativamente el bundle.
* Fetches repetidos.
* Componentes pesados cargados innecesariamente.

---

## 17. Principio específico de React

Priorizar problemas relacionados con:

* Estado mal gestionado.
* Efectos innecesarios o incorrectos.
* Renderizados realmente problemáticos.
* Componentes con responsabilidades excesivas.
* Race conditions.
* Fetches duplicados.
* Problemas de accesibilidad.
* Seguridad del lado del cliente.

No convertir reglas de estilo React en problemas críticos.

La auditoría debe buscar **problemas reales del comportamiento y arquitectura de la aplicación**, no simplemente imponer una determinada forma de escribir componentes.


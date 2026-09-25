# TypeScript Audit Rules

Reglas específicas para proyectos TypeScript.

## 1. Tipado

Revisar:

* Uso innecesario de `any`.
* Tipos excesivamente amplios.
* Tipos incorrectos o engañosos.
* Uso innecesario de `unknown`.
* Interfaces o tipos duplicados.
* Tipos que no representan correctamente el dominio.

Priorizar tipos que mejoren realmente la seguridad y mantenibilidad.

---

## 2. `any`

Detectar:

* `any` innecesarios.
* Propagación de `any` entre funciones.
* APIs importantes sin tipado.
* Castings utilizados para evitar errores del compilador.

No considerar `any` automáticamente como un error. Evaluar su contexto.

---

## 3. Type Assertions

Revisar:

```ts
value as Type
<Type>value
```

Detectar assertions utilizadas para ocultar incompatibilidades reales.

Especial atención a:

```ts
as any
as unknown as Type
```

cuando se utilizan para saltarse el sistema de tipos.

---

## 4. Null Safety

Revisar:

* Uso incorrecto de `!`.
* Posibles valores `null` o `undefined`.
* Comprobaciones redundantes.
* Tipos opcionales utilizados incorrectamente.
* Diferencias entre `null` y `undefined` cuando tengan significado en el dominio.

---

## 5. Interfaces y Types

Detectar:

* Definiciones duplicadas.
* Tipos excesivamente grandes.
* Tipos que representan múltiples conceptos diferentes.
* Interfaces con responsabilidades no relacionadas.
* Tipos que podrían reutilizarse correctamente.

No recomendar convertir automáticamente `interface` en `type` o viceversa.

---

## 6. Generics

Revisar:

* Generics innecesarios.
* Generics excesivamente complejos.
* Restricciones ausentes cuando son necesarias.
* Repetición de tipos que podría evitarse mediante un generic apropiado.

No introducir generics únicamente para reducir unas pocas líneas de código.

---

## 7. Enums y Unions

Evaluar el uso de:

* `enum`
* Union types
* Literal types

Detectar estructuras que podrían representar mejor estados o valores mediante tipos discriminados.

---

## 8. Discriminated Unions

Cuando existan múltiples estados de una misma entidad, revisar si el código:

* Distingue correctamente los estados.
* Maneja todos los casos.
* Evita combinaciones de propiedades inválidas.

Especial atención a estructuras que permiten estados imposibles.

---

## 9. Narrowing

Revisar:

* Type guards incorrectos.
* Narrowing perdido innecesariamente.
* Assertions utilizadas donde podría realizarse narrowing seguro.
* Condiciones que el compilador podría verificar mejor.

---

## 10. Funciones

Analizar:

* Parámetros excesivos.
* Tipos demasiado genéricos.
* Valores de retorno ambiguos.
* Funciones que devuelven diferentes estructuras sin necesidad.
* Callbacks sin tipado adecuado.

---

## 11. `tsconfig`

Revisar la configuración cuando sea relevante.

Considerar:

* `strict`
* `noImplicitAny`
* `strictNullChecks`
* `noUncheckedIndexedAccess`
* `noUnusedLocals`
* `noUnusedParameters`

No exigir una configuración concreta sin considerar la naturaleza y compatibilidad del proyecto.

---

## 12. Errores y excepciones

Detectar:

* `catch` con errores tipados incorrectamente.
* Conversión insegura de errores.
* Assertions sobre errores.
* Pérdida de información del error original.

Considerar el tipo `unknown` cuando corresponda.

---

## 13. Tipos duplicados

Buscar representaciones diferentes del mismo concepto.

Ejemplo:

```text id="9g5vbf"
User
Usuario
UserResponse
UserData
UserDTO
```

Si representan exactamente el mismo concepto sin una razón arquitectónica, evaluar si existe duplicación innecesaria.

Si representan capas diferentes, no considerarlos automáticamente duplicados.

---

## 14. Tipos generados

Identificar tipos generados automáticamente por:

* ORM.
* OpenAPI.
* GraphQL.
* Código generado.
* Herramientas externas.

No modificar ni recomendar cambios manuales sobre código generado.

Analizar la fuente que genera esos tipos cuando sea necesario.

---

## 15. JavaScript dentro de TypeScript

Detectar archivos o zonas TypeScript donde:

* Se evita sistemáticamente el tipado.
* Se utiliza `any` como sustituto de tipos.
* Se realizan múltiples assertions.
* El código prácticamente funciona como JavaScript.

Evaluar si esto representa deuda técnica significativa.

---

## 16. Principio específico de TypeScript

Priorizar problemas relacionados con:

* Errores que el compilador podría detectar.
* Tipos incorrectos del dominio.
* Pérdida de seguridad de tipos.
* Assertions inseguras.
* Estados imposibles representados por los tipos.
* Duplicación significativa de definiciones.

El objetivo no es maximizar la cantidad de tipos, sino utilizar el sistema de tipos para reducir errores y mejorar el diseño.


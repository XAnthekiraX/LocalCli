---
title: SPEC — Ciclo de planificación
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-MOTOR-FLUJOS]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
relacionado:
  - "[[specs/SPEC-CICLO-TRABAJO]]"
---
# SPEC — Ciclo de planificación

Prioridad: P0 (núcleo)

## Propósito

Crear desde cero la documentación de un proyecto que todavía no tiene, para que el agente `build` pueda trabajar con ella.

## Alcance

Incluye el recorrido completo de planificación: visión, requisitos, decisión técnica, documentación por capas, estructura de tareas y validación.
No incluye la ejecución del código, que corresponde a `build` en [[specs/SPEC-CICLO-TRABAJO]].

## Actores

- **Usuario**: responde preguntas, aprueba cada propuesta.
- **Agente `plan`**: pregunta, investiga y propone. No escribe.
- **Agente `build`**: escribe los documentos y las tareas una vez aprobados.
- **Sistema**: aplica las reglas de permiso a cada documento escrito.

## Cuándo se usa

Cuando se abre una carpeta sin documentación, o cuando la documentación existente hay que cambiarla o ampliarla. Si el proyecto ya está documentado y solo hay que ejecutar, este ciclo no aplica.

## Flujo principal

1. **Tipo de proyecto.** `plan` pregunta si es profesional o de prueba. El tipo decide cuánto se documenta: un proyecto de prueba solo necesita lo mínimo para funcionar.
2. **Visión.** `plan` propone qué hace el proyecto, qué problema resuelve y qué no incluye. `build` lo escribe.
3. **Requisitos.** `plan` propone las especificaciones funcionales, una por archivo. Una especificación = una funcionalidad.
4. **Decisión técnica.** `plan` define stack, arquitectura y persistencia, y lo justifica. `build` lo escribe.
5. **Documentación por capas.** `plan` propone un archivo cada vez, y tras tu aprobación `build` lo escribe. **No se pasa al siguiente hasta que el actual esté escrito y aprobado.** El orden es base de datos, luego backend, luego frontend.
6. **Estructura de trabajo.** `plan` propone la lista ordenada de trabajo de la planificación; `build` la escribe.
7. **Validación.** Se revisa que la documentación sea coherente entre capas, que no repita información y que respete la estructura. Se corrige lo que encuentre.
8. **Entrega.** La petición de documentar capa por capa se detecta y crea el TODO con todas las fases de la planificación, y `build` empieza a trabajar sin que hagas nada. Ver [[specs/SPEC-COLA-TAREAS]].

## Orden de las capas

```
base de datos  →  backend  →  frontend
```

El orden no es arbitrario: el backend documenta sus endpoints contra lo que la base de datos ya definió, y el frontend documenta sus pantallas contra los endpoints del backend. Escribirlo al revés obliga a repetir datos y luego a corregirlos.

## Flujos alternativos

- El proyecto ya tiene documentación: `plan` compara, propone qué falta y solo se escribe eso.
- El proyecto es de prueba: se reduce al mínimo de cada capa.
- El usuario rechaza un documento: se revisa ese archivo y se vuelve a pedir aprobación.
- Falta información: se avisa y la fase se detiene.
- Una capa no aplica: se omite y queda registrado que no aplica.

## Reglas de negocio

- Las capas se documentan en orden: base de datos, backend, frontend.
- Nada se escribe sin que el usuario apruebe el archivo anterior.
- Máximo 5 preguntas relacionadas por tanda.
- Un archivo = una responsabilidad.
- La información no se duplica: si ya está escrita en otro archivo, se referencia.
- Las referencias entre archivos usan el formato de enlace.
- Cada capa tiene un archivo principal en su raíz que centraliza su información. No hay un archivo de entrada aparte.
- Las carpetas usan prefijo numérico por jerarquía.
- La documentación se escribe en español. Los nombres técnicos, identificadores y código pueden quedar en inglés.
- Un recurso = un archivo. Los DTOs van aparte, uno por recurso.
- `plan` no escribe. Propone, tú apruebas y `build` escribe.
- `plan` nunca escribe código.
- El progreso de cada fase queda registrado en un archivo de seguimiento.
- Al terminar, la documentación está completa y validada.
- Al terminar, la cola se puebla sola y `build` arranca. La planificación no espera a que el usuario lance nada.

## Criterios de aceptación

- [ ] Al abrir una carpeta vacía, `plan` propone empezar el ciclo de planificación.
- [ ] Pregunta primero si el proyecto es profesional o de prueba.
- [ ] Las capas se documentan en orden: base de datos, backend, frontend.
- [ ] Un documento se propone, se aprueba y `build` lo escribe antes de pasar al siguiente.
- [ ] `plan` no escribe ningún archivo durante la planificación.
- [ ] Nunca hace más de 5 preguntas por tanda.
- [ ] Un archivo contiene una sola responsabilidad.
- [ ] La misma información no aparece en dos archivos: se referencia.
- [ ] Cada capa tiene su archivo principal en la raíz y las carpetas llevan prefijo numérico.
- [ ] Al terminar existe el TODO con todas las fases de la planificación, en orden.
- [ ] La validación detecta incoherencias entre capas y referencias rotas, y se corrigen.
- [ ] Al terminar, existe el TODO de la planificación y `build` ya está trabajando.
- [ ] Al terminar, `build` puede trabajar con la documentación sin preguntar nada básico.
- [ ] Ningún documento se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.

## Requisitos no funcionales

- La documentación resultante no repite la misma información en varios archivos.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-TOOLS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-MOTOR-FLUJOS]]
- [[specs/SPEC-COLA-TAREAS]]

## Supuestos

- La estructura exacta de carpetas y los nombres de los archivos por capa se fijan en FASE 2 y FASE 3.
- Las etapas internas de la decisión técnica y del diseño quedan por definir.

## Referencias

- [[IDEA]]

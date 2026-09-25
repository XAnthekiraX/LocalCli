# SPEC — Nodo de contexto

Prioridad: P0 (núcleo)

## Propósito

Decidir qué pequeña parte de la documentación del proyecto se le entrega al modelo, para que no cargue con todo.

## Alcance

Incluye recorrer las dependencias declaradas en la documentación, dejar que el modelo decida qué es relevante, recortar lo que sobra y registrar qué entró y qué salió.
No incluye cómo se usa dentro de un flujo, que está en [[specs/SPEC-MOTOR-FLUJOS]].

## Actores

- **Etapa del flujo**: pide contexto para un objetivo concreto.
- **Modelo**: decide qué es relevante.
- **Sistema**: lee, recorta, entrega y registra.
- **Usuario**: puede consultar qué se descartó.

## Flujo principal

1. Una etapa pide contexto para un objetivo, por ejemplo "implementar la API de pedidos".
2. El sistema lee los documentos del proyecto y las dependencias que cada uno declara.
3. Se le pide al modelo que diga qué documentos necesita para ese objetivo.
4. El modelo devuelve la lista.
5. El sistema carga solo esos documentos y recorta lo que sobre.
6. Se entrega el contexto a la etapa siguiente.
7. Se registra qué documentos entraron, cuáles salieron y por qué.

## Ejemplo

Un documento de APIs declara que usa los DTOs de respuesta. El documento de esos DTOs declara que usa las entities. El harness no lee el proyecto entero: parte del documento de APIs, sigue las declaraciones y llega solo a lo que hace falta.

## Flujos alternativos

- El modelo pide un documento que no existe: se informa y la etapa se detiene.
- Hay documentos sin dependencias declaradas: se avisa para que se completen.
- Un documento es demasiado grande: se recorta por sección.
- El modelo no puede decidir: se usa por defecto el objetivo declarado por la etapa.

## Reglas de negocio

- La documentación declara sus dependencias en el frontmatter mediante enlaces. Ese mapa sirve para orientarse, no decide por sí solo.
- La decisión de qué es relevante la toma siempre el modelo.
- El contexto entregado nunca supera el límite de contexto del modelo.
- Todo lo que se descartó queda registrado con el motivo.
- El nodo de contexto no lee fuera de la carpeta del proyecto sin permiso.
- Un documento que entra en el contexto se lee primero; no se entrega nada sin leer.
- El contexto de una etapa no arrastra el de etapas anteriores si no lo necesita.

## Criterios de aceptación

- [ ] Dada una etapa con un objetivo, el modelo recibe solo los documentos que declaró necesarios, no el proyecto completo.
- [ ] Si un documento declara que usa otros, el sistema los encuentra siguiendo esas declaraciones.
- [ ] El modelo decide qué es relevante aunque el mapa de dependencias exista.
- [ ] El registro indica qué documentos entraron, cuáles salieron y por qué.
- [ ] Si el modelo pide algo que no existe, la etapa se detiene y avisa en vez de continuar con suposiciones.
- [ ] El contexto entregado cabe dentro del límite del modelo.
- [ ] El usuario puede consultar el registro de selección de cualquier etapa.

## Requisitos no funcionales

- El recorte reduce de forma medible el tamaño del contexto entregado frente a leer el proyecto completo.
- Preparar el contexto no debe tardar más que la propia respuesta del modelo.

## Dependencias funcionales

- [[specs/SPEC-OLLAMA-PERFIL]]
- [[specs/SPEC-MOTOR-FLUJOS]]
- [[specs/SPEC-ARCHIVOS]]

## Supuestos

- El formato exacto de la declaración de dependencias en el frontmatter se fija en FASE 2 y FASE 3.
- El motivo del descarte lo genera el modelo durante la selección.

## Referencias

- [[IDEA]]

---
name: generador-publicaciones
description: >
  Analiza un proyecto y genera un único archivo Publicaciones.md con una serie de
  publicaciones ordenadas cronológicamente. Cada publicación representa una etapa
  importante del desarrollo utilizando un enfoque de AI Native Development.
---

# Objetivo

Analizar el proyecto y reconstruir su proceso de desarrollo.

No documentar el sistema.

No escribir las publicaciones.

Solo generar la estructura y el contexto necesario para redactarlas posteriormente.

---

# Análisis

Analizar todo el proyecto.

Priorizar:

- TASK.md
- Documentación
- Arquitectura
- Reglas de negocio
- Código
- Base de datos
- APIs
- Componentes
- Configuración

Nunca inventar información.

---

# Salida

Generar únicamente:

Publicaciones.md

---

# Reglas

- La Publicación 1 siempre será la recopilación del contexto inicial del proyecto.
- Su contenido deberá quedar vacío porque esa información normalmente no puede deducirse del código.
- Desde la Publicación 2 el agente decidirá automáticamente el orden de las publicaciones.
- No utilizar una lista fija.
- Agrupar los cambios en etapas lógicas del desarrollo.
- Utilizar títulos generales, no detalles de implementación.
- Cada publicación debe depender de la anterior.

---

# Formato

# Publicación X

## Título

## Descripción

Resumen muy corto de la etapa.

## Contexto encontrado

Síntesis de todo lo encontrado relacionado con esa etapa.

## Proceso realizado por la IA

Resumen de cómo agrupó la información y por qué considera que pertenece a esa etapa.

---

# Ejemplo

# Publicación 1

## Título

Origen del proyecto

## Descripción

Contexto inicial del proyecto.

## Contexto encontrado

(Completar manualmente.)

## Proceso realizado por la IA

No aplica.

---

# Publicación 2

## Título

Definición de necesidades del sistema

## Descripción

Análisis de los requerimientos identificados.

## Contexto encontrado

- ...
- ...
- ...

## Proceso realizado por la IA

- Analizó TASK.md.
- Analizó las reglas de negocio.
- Relacionó la documentación funcional.
- Agrupó toda la información en una única etapa del desarrollo.


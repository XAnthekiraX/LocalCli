---
title: LocalCli — pruebas de la TUI
tags: [frontend, calidad]
depende_de:
  - "[[backend/05-quality/TESTING]]"
  - "[[frontend/FRONTEND]]"
relacionado:
  - "[[frontend/02-interfaces/INTERFACES]]"
  - "[[database/01-schema/ENUMS]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# TESTING — Pruebas de la TUI

Qué se prueba en la capa de presentación y cómo. La estrategia global está en [[backend/05-quality/TESTING]]; esto la complementa para la interfaz.

## 1. Estrategia

| Nivel | Qué cubre | Cómo |
|---|---|---|
| Unitario | Funciones puras de render: recorte de líneas, formato del bloque de razonamiento, formato del contador, detección de atajos duplicados | Funciones aisladas, sin bucle de Bubble Tea |
| De componente | `update` y `view` de cada componente ante secuencias fijas de eventos | Bubble Tea en proceso, con el arnés de pruebas de la librería, inyectando mensajes sintéticos |
| De integración | El recorrido completo: tecla → petición a `session` → evento de vuelta → pantalla | `session` real con base temporal y un doble de `ollama` que emita tokens fijos |

## 2. Qué debe probarse

- El panel muestra los nueve datos definidos en [[specs/SPEC-INTERFAZ]] y refleja la sesión activa, no otra.
- El razonamiento se distingue de la respuesta, se muestra en vivo y se oculta sin detener la generación.
- Con el panel cerrado, el contador de aprobaciones pendientes sigue visible y se actualiza con cada evento.
- El selector lista todas las sesiones con su estado, incluidas las de segundo plano; elegir una cambia el chat sin detener nada.
- El panel de aprobaciones lista pendientes de cualquier sesión y resuelve cada línea por separado.
- Los estados pintados coinciden con [[database/01-schema/ENUMS]]; ningún estado inventado.
- Un atajo duplicado se rechaza al guardar el mapa de teclas, y el cambio queda persistido.
- Si la sesión activa está generando, la entrada sigue operativa y no cancela nada.
- Al ejecutar la aplicación se ve la bienvenida: logotipo ASCII, nombre con versión y una línea de entrada; sin panel, selector ni aprobaciones.
- La primera petición escrita en la bienvenida llega a `session` y aparece como primer mensaje del chat al cambiar de vista, sin repetirse ni pedir confirmación.
- La bienvenida se pinta sin Ollama ni base: se comprueba con ambos no disponibles.
- El logotipo se compara byte a byte contra la salida dorada `internal/tui/testdata/logo.txt`: 6 filas × 53 columnas, arte fijo, sin variaciones. La definición canónica está en [[specs/SPEC-INTERFAZ]].

## 3. Reglas para nuevos tests

- **Las vistas se comparan contra salidas doradas** (golden files) en lo que a formato respecta: un cambio de estilo se revisa a la vista, no a ciegas.
- **No hay `sleep` ni esperas fijas.** Los eventos se inyectan y se espera la condición, no un tiempo.
- **Una prueba de componente no toca la base ni la red.** Los eventos son valores, no llamadas; el doble de `ollama` solo entra en las pruebas de integración.
- **Una prueba, una regla**, igual que en el backend: el nombre dice la regla que comprueba.
- Los criterios de aceptación de [[specs/SPEC-INTERFAZ]] y [[specs/SPEC-INTERFAZ-ATAJOS]] son la lista de verificación manual final.

## Referencias

- [[backend/05-quality/TESTING]] — estrategia global y dobles de Ollama.
- [[frontend/02-interfaces/INTERFACES]] — la frontera que estas pruebas cubren.
- [[specs/SPEC-INTERFAZ]] — criterios de aceptación de la disposición.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — criterios de aceptación de atajos y aprobaciones.

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
    - "[[specs/SPEC-VISTA-CODIGO]]"
---

# TESTING — Pruebas de la TUI

Qué se prueba en la capa de presentación y cómo. La estrategia global está en [[backend/05-quality/TESTING]]; esto la complementa para la interfaz.

## 1. Estrategia

| Nivel          | Qué cubre                                                                                                                                                               | Cómo                                                                                          |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Unitario       | Funciones puras de render: recorte de líneas, formato del bloque de razonamiento, formato del contador, formato del tiempo de respuesta, detección de atajos duplicados | Funciones aisladas, sin bucle de Bubble Tea                                                   |
| De componente  | `update` y `view` de cada componente ante secuencias fijas de eventos                                                                                                   | Bubble Tea en proceso, con el arnés de pruebas de la librería, inyectando mensajes sintéticos |
| De integración | El recorrido completo: tecla → petición a `session` → evento de vuelta → pantalla                                                                                       | `session` real con base temporal y un doble de `llm` que emita tokens fijos |

## 2. Qué debe probarse

- El panel muestra los nueve datos definidos en [[specs/SPEC-INTERFAZ]] y refleja la sesión activa, no otra.
- El panel muestra la lista de pasos del agente (`todo_actualizada`) con `[•]`/`[✓]`/`[ ]`, la oculta cuando no queda nada accionable y no muestra la lista de otra sesión.
- Mientras el modelo trabaja se ve el indicador en vivo (`Pensando`, `Usando herramienta: X`, `Generando`); el razonamiento no se vuelca por defecto y `Ctrl+R` revela su texto sin detener la generación.
- El contador de tokens del turno se muestra bajo la entrada y el valor exacto (`tokens_turno`) corrige la aproximación viva; el formato del conteo es una función pura (entero, miles con decimal, miles enteros).
- Cada respuesta lleva su tiempo de llegada y el contador en vivo corre mientras el turno está en curso y se detiene al cerrarse; el formato de la duración es una función pura (milisegundos, segundos, minutos y segundos).
- Con el panel cerrado, el contador de aprobaciones pendientes sigue visible y se actualiza con cada evento.
- `Ctrl+X l` abre el modal de sesiones con todas las del proyecto y su estado, incluidas las de segundo plano; elegir una con `Enter` abre esa sesión (el chat cambia a su historial) sin detener nada.
- `Ctrl+P` abre el modal de atajos: cada línea muestra la acción y su tecla; es de solo lectura. `Esc` cierra los cuatro modales sin cambiar nada.
- `Tab` alterna el agente `plan`/`build`; el indicador `[plan]`/`[build]` aparece a la izquierda del input en bienvenida y vista principal, y no cicla con un modal abierto.
- Escribir `/` despliega la paleta de comandos de flujo encima de la entrada —también en la bienvenida—; el filtro y las flechas la recorren, `Tab` autocompleta el comando resaltado y `Enter` lo ejecuta. El catálogo sale del puerto (oficiales sembrados en `~/.config/localcli/flows/` y propios de `<proyecto>/.localcli/flows/`) y lo que no es comando, aunque empiece por `/`, se responde como chat.
- El panel de aprobaciones lista pendientes de cualquier sesión y resuelve cada línea por separado.
- Los estados pintados coinciden con [[database/01-schema/ENUMS]]; ningún estado inventado.
- Un atajo duplicado se rechaza al guardar el mapa de teclas, y el cambio queda persistido.
- Si la sesión activa está generando, la entrada sigue operativa y no cancela nada.
- Detener el turno con doble `esc` conserva lo que el modelo ya entregó: lo parcial se queda en pantalla y se guarda con su tiempo, para que la sesión retomada lo muestre.
- La línea de entrada salta de renglón al desbordar el ancho —sin recortar lo escrito— y reajusta el reparto al redimensionar la terminal; `Enter` envía y no inserta saltos. La cuenta de filas envueltas y el reparto de la bienvenida son funciones puras.
- La selección con el ratón se limita a la zona donde empezó el arrastre —chat, sidebar o caja de entrada— y no mezcla zonas: un arrastre que empieza en la caja de entrada no alcanza el logotipo, las pistas ni la versión. La selección está anclada al texto: un repintado de la vista (la respuesta generándose token a token) no la cancela ni la desplaza a otro texto, y el realce sigue al texto elegido al rehacerse el marco.
- Al pegar o arrastrar —tanto en la caja de la bienvenida como en la de la vista principal—, la entrada muestra cada archivo como `[nombre.ext]` y cada carpeta como `[CARPETA N elementos]` (un texto de varias líneas como `[PEGADO N líneas]`); al enviar llega el valor real (la imagen se adjunta, el texto entero viaja). Un pegado que llega como `KeyRunes` con `Paste` en alto (bracketed paste, `Ctrl+Shift+V`) se tokeniza; lo tecleado letra a letra se escribe tal cual.
- En el chat, lo escrito por el usuario y lo que responde el agente se pintan en globos de color distinto; las líneas del sistema (herramientas, avisos) van sueltas.
- La línea de herramienta nace al invocar con su verbo y su tema (la ruta que se busca) y se completa al llegar el resultado con la marca, la medida («70 líneas»), el aviso de recorte si lo hubo y el tiempo que tardó la ejecución; es una sola línea, no nombra al agente y no vuelca la salida cruda. El tiempo se pinta también cuando la ejecución falló, y si no se midió no se pinta nada.
- La línea de herramienta cerrada se pinta con su duración al recargar la sesión, igual que en vivo: el hilo recuperado no pierde los tiempos.
- Bajo la línea de `editar_archivo`, `escribir_archivo` y `crear_archivo` se pinta el bloque de diferencia con las líneas quitadas (`-`) y añadidas (`+`); un cambio fallido no deja bloque; un diff que excede el tope cierra con `… N líneas más`; el bloque se recupera con la línea al retomar la sesión y no entra al contexto. La comparación de estructura es una salida dorada, sin depender de colores.
- La duración de una línea de herramienta y la de su respuesta salen del mismo formateador —milisegundos por debajo del segundo, décimas de segundo por debajo del minuto, minutos y segundos a partir de ahí— con el mismo atenuado; y una duración ausente no produce ni un `0` ni un hueco reservado.
- El chat respeta el orden de ejecución: el texto que precede a una herramienta queda en un globo arriba de su línea y el que viene después abre un globo nuevo, con su propio razonamiento; los segmentos de texto de un turno no se funden en uno solo.
- El chat interpreta el Markdown de la respuesta al pintar ([[specs/SPEC-VISTA-CODIGO]]): un bloque de código entre fences se pinta como superficie propia con su cabecera y su lenguaje, sin mostrar los fences; las líneas largas se envuelven; el Markdown del cuerpo no se interpreta; títulos, negrita, cursiva, código inline, listas, citas y reglas se pintan con su estilo y el resto queda literal. El parser y el render del subconjunto son funciones puras.
- Un clic sobre el botón de copiar de la cabecera de un bloque copia el cuerpo exacto del bloque (sin fences ni cabecera) al portapapeles y muestra `[Copiado]`; el clic no inicia una selección. La copia usa el doble de portapapeles en las pruebas.
- Al ejecutar la aplicación se ve la bienvenida centrada: logotipo ASCII, nombre con versión, la línea con el modelo en uso y una línea de entrada; sin panel de contexto, sin aprobaciones y sin lista de modelos visible. `Ctrl+X m` abre el modal con los modelos que sirve `llama-server`, mostrados por su nombre —la carpeta en `models/`— (aviso «sin modelos» si no responde); `↑`/`↓` navegan, `Enter` aplica y viaja con la primera petición, `Esc` cierra sin cambios.
- La primera petición escrita en la bienvenida llega a `session` y aparece como primer mensaje del chat al cambiar de vista, sin repetirse ni pedir confirmación.
- La bienvenida se pinta sin motor de inferencia ni base: se comprueba con ambos no disponibles.
- El logotipo se compara byte a byte contra la salida dorada `internal/tui/testdata/logo.txt`: 6 filas (anchos 69, 69, 66, 66, 69, 69 columnas), arte fijo, sin variaciones. El compacto (`internal/tui/testdata/logo-compacto.txt`, 6 filas, anchos 39, 39, 36, 36, 39, 39 columnas, `L.CLI`) también se compara byte a byte. La definición canónica está en [[specs/SPEC-INTERFAZ]].
- La bienvenida se prueba a distintos anchos: por debajo de 70 columnas pinta el logotipo compacto y ninguna fila supera el ancho de la pantalla.
- La línea de estado de la caja —en la bienvenida y en la principal— envuelve sus chapas en vez de recortarlas cuando no caben, de modo que el interruptor `pensar` sigue visible y su rectángulo sigue localizable para el ratón; las líneas del panel de aprobaciones conservan siempre `aprobar | declinar` en cualquier ancho.

## 3. Reglas para nuevos tests

- **Las vistas se comparan contra salidas doradas** (golden files) en lo que a formato respecta: un cambio de estilo se revisa a la vista, no a ciegas.
- **No hay `sleep` ni esperas fijas.** Los eventos se inyectan y se espera la condición, no un tiempo.
- **Una prueba de componente no toca la base ni la red.** Los eventos son valores, no llamadas; el doble de motor solo entra en las pruebas de integración.
- **Una prueba, una regla**, igual que en el backend: el nombre dice la regla que comprueba.
- Los criterios de aceptación de [[specs/SPEC-INTERFAZ]] y [[specs/SPEC-INTERFAZ-ATAJOS]] son la lista de verificación manual final.

## Referencias

- [[backend/05-quality/TESTING]] — estrategia global y dobles de motor.
- [[frontend/02-interfaces/INTERFACES]] — la frontera que estas pruebas cubren.
- [[specs/SPEC-INTERFAZ]] — criterios de aceptación de la disposición.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — criterios de aceptación de atajos y aprobaciones.

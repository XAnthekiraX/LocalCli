---
title: LocalCli — mejoras de la TUI
tags: [frontend, mejoras]
depende_de:
    - "[[frontend/FRONTEND]]"
relacionado:
    - "[[frontend/01-domain/DOMAIN]]"
    - "[[frontend/05-quality/TESTING]]"
---

# MEJORAS — TUI

Revisión de la capa de interfaz contra cuatro reglas de estructura:

1. Cada archivo tiene **una responsabilidad**.
2. El código se **separa por carpetas con nombres descriptivos**.
3. El código de cada responsabilidad **no es largo**; si lo es, se parte en
   responsabilidades más pequeñas.
4. El resultado es **entendible y escalable**.

Este documento no cambia código: registra los incumplimientos, la estructura
objetivo y el plan para abordarlos. La arquitectura vigente sigue en
[[frontend/FRONTEND]] y [[frontend/01-domain/DOMAIN]].

## 1. Resumen ejecutivo

| Regla                              | Estado | Nota                                                                                 |
| ---------------------------------- | ------ | ------------------------------------------------------------------------------------ |
| Una responsabilidad por archivo    | Parcial | Los componentes la cumplen; `app.go` y la raíz del paquete no                      |
| Carpetas con nombres descriptivos  | Parcial | `components/` y `render/` sí; teclado, paleta y selección viven planos en la raíz  |
| Código de responsabilidad no largo | No     | `app.go` (923 líneas) concentra al menos 8 responsabilidades                       |
| Entendible y escalable             | Parcial | La base es sana; la raíz `tui` mezcla capas y frena el crecimiento                  |

**Lo que ya cumple.** `components/` usa nombres descriptivos (`welcome`,
`input`, `chat`, `panel`, `modals`) y hay una capa `render/` compartida. La
decomposición interna de `chat/` (modelo, turno, líneas, formato, razonamiento,
vista), `panel/` (modelo, formato, vista), `modals/` (mecánica común + uno por
tipo) e `input/` (modelo + adjuntos) es correcta.

## 2. Hallazgos por severidad

### 2.1 Crítico

**H1 — `internal/tui/app.go` (923 líneas): god-object.** Concentra:

- modelo raíz y constructor (`App`, `Nueva`, `Init`, `Update`, `View`);
- render de vistas (`marco`, `vistaPrincipal`, `vistaBienvenida`, `vistaModal`,
  `conSeleccion`, `componerColumnas`, `anchoPanel`, `anchoColumna`, `altoDe`);
- gestión de modales (`modalAbierto`, `contextoModal`, `soloModal`,
  `cerrarModales`, `cerrarModal`, `moverModal`, `aplicarModal`,
  `abrirModalModelos`, `abrirModalSesiones`, `alternarModalAtajos`,
  `aplicarModelo`, `eliminarResaltada`, `trasEliminarSesion`, `pedirModelos`,
  `pedirSesiones`, `eliminarSesionCmd`, `modelosDeLocal`, `sesionesDeLocal`);
- sesión y latido (`adoptarSesion`, `sesionID`, `reiniciarPanel`, `latido`,
  `arrancarLatido`);
- preferencias y teclas (`guardarPreferencias`, `cargarPreferencias`,
  `cargarTeclas`);
- agentes (`ciclarAgente`, `posicionAgente`, `tieneAgente`);
- despacho de acciones (`despachar`);
- tipos internos (`Vista`, `ModalVista`, `frameMsg`, `modelosMsg`, `sesionesMsg`,
  `sesionEliminadaMsg`).

Viola la regla 3: una responsabilidad larga debe partirse.

**H2 — La paleta de comandos no está en `components/`.** `internal/tui/comandos.go`
(251 líneas) es un componente completo (estado `PaletaModel`, filtro, resaltado y
`View`) y vive plano en la raíz, mientras el resto de componentes están bajo
`components/`. Viola la regla 2.

### 2.2 Alto

**H3 — El subsistema de teclado vive plano en la raíz.** `keymap.go` (445),
`keyresolver.go` (274) y `keys.go` (148) suman 867 líneas de una misma
responsabilidad («mapa y resolución de teclas») sin carpeta propia.

**H4 — `internal/tui/selection.go` (349 líneas) mezcla cuatro cosas:** geometría
(`punto`, `rect`), la máquina de selección, la copia al portapapeles
(`copiarAlPortapapeles`, OSC 52) y el enrutado del ratón (`raton`) junto con el
indicador superior (`conIndicadorSuperior`). Además, con la selección del chat en
**coordenadas de contenido** y el **desplazamiento automático** al arrastrar contra
un borde (selección que rebasa el viewport), la máquina de selección suma una
responsabilidad más: el chat necesita exponer la traducción contenido↔ventana.

**H5 — Edición de línea duplicada.** `components/welcome/welcome.go` y
`components/input/input.go` repiten idénticos `Escribir`, `Borrar`,
`BorrarAdelante`, `Izquierda`, `Derecha`, `Inicio`, `Fin` y `acotar` (~80
líneas). El pintado del cursor (`textoConCursor`) también se repite en
`welcome/caja.go` e `input/input.go`.

**H6 — Datos de demo en código de producción.** `app.go:859 cargarMuestraChat`
inyecta burbujas de prueba en el constructor de la aplicación.

### 2.3 Medio

**H7 — Helpers duplicados.** Dos rectángulos (`rect` y `input.Rect`); `acotar` en
`selection.go`, `render/texto.go` y `chat/vista.go`; `cuentaLineas` (welcome) y
`altoDe` (app) son el mismo cálculo; `recortar` (comandos) solapa con
`truncar`/`RecortarColumnas` (render). `truncar`, `envolverPorRuta` y
`corteAncho` (panel) son genéricos y encajan mejor en `render/`.

**H8 — Pistas de la bienvenida hardcodeadas.** `welcome/pistas.go` fija los
atajos en un `var` con el comentario «el mapa central de teclas los generará
aquí», pero [[frontend/01-domain/DOMAIN]] dice que la barra se genera desde el
mapa vigente.

**H9 — Documentación desincronizada.** [[frontend/FRONTEND]] §2 describe la
estructura antigua (plana: `welcome.go`, `chat.go`, `styles.go`, `marco.go`…) y no
refleja `components/` ni `render/`.

## 3. Estructura objetivo propuesta

```
internal/tui/
├── app.go              → modelo raíz, Nueva, Init, Update, View (solo enrutado)
├── vista.go            → marco, vistas, componerColumnas, anchos, altoDe
├── modales.go          → apertura/cierre/aplicación de modales y sus lecturas
├── sesion.go           → adopción/refresco de sesión y latido del indicador
├── preferencias.go     → carga/guarda de preferencias y teclas
├── agentes.go          → ciclado y validación de agentes
├── acciones.go         → despachar
├── mensajes.go         → Vista, ModalVista y mensajes internos
├── arranque.go         → Correr (bootstrap)
├── wire.go             → Puerto (frontera con el orquestador)
├── logo.go / doc.go
├── components/
│   ├── paleta/         → comandos.go movido aquí
│   ├── welcome/  input/  chat/  panel/  modals/
├── keys/               → keymap.go, keyresolver.go, keys.go
├── selection/          → geometria.go, seleccion.go, portapapeles.go, raton.go
└── render/             → estilos, marco, texto (incluye helpers genéricos)
```

## 4. Plan de refactor por fases

| Fase | Alcance                                                                                                     | Riesgo |
| ---- | ----------------------------------------------------------------------------------------------------------- | ------ |
| 1    | Partir `app.go` en `vista.go`, `modales.go`, `sesion.go`, `preferencias.go`, `agentes.go`, `acciones.go`, `mensajes.go` (mismo paquete `tui`) | Bajo   |
| 2    | Crear `components/paleta/`, `keys/` y `selection/`; exportar los símbolos necesarios y ajustar llamadas   | Medio  |
| 3    | Extraer un editor de línea compartido (`welcome` + `input`); mover helpers genéricos a `render/`; sacar `cargarMuestraChat` a tests | Medio  |
| 4    | Generar `welcome/pistas.go` desde el keymap; actualizar [[frontend/FRONTEND]] §2 y [[frontend/01-domain/DOMAIN]] | Bajo   |

**Quick-fix ya acordado (H6).** Quitar `cargarMuestraChat` del constructor de
producción: eliminar la llamada en `Nueva` y el método, y conservar el estado de
demo como helper de tests en la Fase 3.

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa y estructura de referencia.
- [[frontend/01-domain/DOMAIN]] — componentes y sus límites.
- [[frontend/05-quality/TESTING]] — estrategia de pruebas.

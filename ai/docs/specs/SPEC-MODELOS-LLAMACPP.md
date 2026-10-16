---
title: SPEC — Modelos con llama.cpp
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[PROJECT]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[backend/04-infrastructure/CONFIGURATION]]"
  - "[[database/01-schema/TABLES]]"
---
# SPEC — Modelos con llama.cpp

Prioridad: P0 (núcleo)

## 1. Objetivo

Utilizar exclusivamente llama.cpp como motor de inferencia local. LocalCLI deberá gestionar las descargas, el almacenamiento y la selección de modelos, además de comunicarse con el servidor mediante una API HTTP.

## 2. Configuración global

Utilizar el directorio existente:

`/home/anthekira/.config/localcli/`

Crear el archivo `localcli.json` con una configuración mínima:

```json
{
  "proveedor": "llama.cpp",
  "api": "http://localhost:8080"
}
```

Este archivo únicamente almacenará el proveedor y la dirección base de la API. No deberá contener el modelo seleccionado ni información sobre los modelos descargados.

## 3. Almacenamiento de modelos

Crear el directorio:

`/home/anthekira/.config/localcli/models/`

Todos los modelos GGUF gestionados por LocalCLI deberán almacenarse en este directorio. Los modelos se cargarán directamente desde sus archivos originales, sin crear copias adicionales para su ejecución.

Cada modelo se guarda en su propio subdirectorio `models/<modelo>/`, con un solo GGUF original por modelo:

```
~/.config/localcli/models/<modelo>/
```

**La carpeta en la que reside el modelo es el nombre del modelo.** Esa carpeta es su identidad y su nombre mostrado en la interfaz: una ruta completa nunca se expone como nombre. La ruta del GGUF es un detalle interno de arranque, no un identificador del modelo.

## 4. Descarga de modelos

Permitir descargar modelos compatibles desde una URL directa de un archivo GGUF (p. ej. `https://huggingface.co/<org>/<repo>/resolve/main/modelo.gguf`). La interfaz expone un modal «DESCARGAR MODELOS» (`/descargar` o `<leader>d`) con un campo de URL y una barra de progreso en vivo; el nombre del modelo se deriva del archivo (sin `.gguf`) y se muestra mientras se escribe.

LocalCLI gestiona la descarga con reanudación (`.part`), validación de tamaño y, si la URL es de Hugging Face, resolución automática del hash SHA256 vía la API del repositorio. El archivo se guarda en `models/<nombre>/<archivo>` y se registra en `models.json`; una descarga en curso se cancela al lanzar otra o al cerrar el modal con `Esc`.

## 5. Catálogo de modelos

LocalCLI deberá mantener un catálogo independiente de la configuración global para registrar los modelos descargados, sus rutas, sus repositorios de origen y los metadatos necesarios para identificarlos.

Este catálogo podrá almacenarse en `models.json`, dentro del directorio global.

El `id` de cada entrada del catálogo es el **nombre del modelo**: el nombre de su subdirectorio en `models/` (sección §3). El catálogo lista, nombra y selecciona modelos solo por ese `id` (el nombre de la carpeta); las rutas de los GGUF quedan en `ruta` como detalle interno.

El modelo seleccionado no deberá agregarse a `localcli.json`. LocalCLI gestionará la selección mediante el catálogo (`seleccionado` = `id` del modelo activo) y el estado de la aplicación.

## 6. Ejecución de modelos

Utilizar `llama-server` para cargar el modelo seleccionado y exponerlo mediante una API HTTP.

LocalCLI deberá poder iniciar y detener el servidor, configurar el puerto y comprobar su disponibilidad.

Al seleccionar un modelo, LocalCLI utilizará la ruta de su archivo GGUF para iniciar el servidor con los parámetros correspondientes. No deberá copiar el archivo a otra ubicación.

La **ventana de contexto** con la que se arranca el servidor (`-c`) sale del **contexto declarado del modelo**, leído de su GGUF (preferentemente `<arch>.context_length` sobre `general.context_length`) al registrarlo en el catálogo, con un **margen del 80 %** para dejar espacio para la respuesta; `LOCALCLI_CONTEXT_LIMIT` la fuerza si está definido (sin margen), y si el modelo no declara contexto se usa un valor por defecto. La ventana efectiva que respeta el turno se confirma luego con el servidor (`props`). Ver [[backend/DECISIONS]] y [[backend/04-infrastructure/INTEGRATIONS]].

Al arrancar el servidor con un modelo, LocalCLI debe identificarlo a `llama-server` **por el nombre del modelo (su carpeta)**, no por su ruta desnuda: el `id` que el servidor publica en `/v1/models` y el nombre que acepta en las peticiones han de ser el nombre de la carpeta (por ejemplo, con el alias correspondiente en el arranque). Así la lista de modelos y las peticiones del chat nombran siempre al modelo igual que la interfaz.

Si el servidor requiere reiniciarse para cargar otro modelo, LocalCLI deberá gestionar ese proceso.

Si el modelo es multimodal, LocalCLI deberá arrancar `llama-server` con el **proyector multimodal** (`--mmproj`) que acompaña al modelo: el archivo `mmproj*.gguf` de su carpeta, o el registrado en su entrada del catálogo (`mmproj`). Sin proyector, el modelo funciona como texto y el servidor no procesa imágenes; con él, el motor declara la visión en `/props` y LocalCLI puede adjuntar imágenes al turno ([[backend/04-infrastructure/INTEGRATIONS]]).

## 7. Comunicación con el harness

Implementar un cliente HTTP para comunicarse con llama-server utilizando la dirección definida en `localcli.json`.

El cliente deberá enviar solicitudes de generación, recibir respuestas y gestionar errores de conexión. La lógica de comunicación permanecerá separada de la interfaz y del procesamiento de las respuestas.

Las solicitudes de generación nombran al modelo por su **nombre (carpeta)**, nunca por la ruta de su archivo. Si lo que reporta el motor al listar es una ruta, el harness la resuelve al nombre de la carpeta que contiene el GGUF; el núcleo común del protocolo no conoce la disposición de `models/`.

## 8. Validaciones

Comprobar que los archivos GGUF existen antes de iniciar el servidor, que las descargas finalizaron correctamente y que el servidor está disponible antes de enviar solicitudes.

Gestionar los errores sin eliminar modelos existentes ni sobrescribir la configuración del usuario.

## 9. Criterio de finalización

La implementación estará completa cuando LocalCLI pueda descargar modelos compatibles desde Hugging Face, almacenarlos en el directorio global, registrarlos en el catálogo, seleccionarlos y ejecutarlos mediante llama.cpp.

La configuración global contendrá exclusivamente el proveedor y la URL de la API. El catálogo de modelos y la lógica de selección se administrarán por separado.

No se utilizará Ollama ni se mantendrán almacenes duplicados de modelos.

## Dependencias funcionales

- [[IDEA]]
- [[PROJECT]]

## Referencias

- [[IDEA]] — el problema y el alcance del proyecto.
- [[PROJECT]] — decisiones técnicas globales.
- [[backend/DECISIONS]] — la decisión que revoca el registro de motores.
- [[backend/04-infrastructure/CONFIGURATION]] — `localcli.json`, `models.json` y el catálogo en la práctica.
- [[database/01-schema/TABLES]] — dónde vive el estado de la aplicación y la selección.
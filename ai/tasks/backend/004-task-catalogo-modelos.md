# 004-task-catalogo-modelos.md

> T-B004 — Catálogo de modelos y selección.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-MODELOS-LLAMACPP]] §3 Almacenamiento de modelos (la carpeta es el nombre), §5 Catálogo de modelos, §8 Validaciones
- [[backend/04-infrastructure/CONFIGURATION]] §8 El catálogo de modelos
- [[backend/DECISIONS]] — la selección vive en el catálogo y el estado de la aplicación

## Contexto

El catálogo es un registro **independiente de la configuración global**: vive en `~/.config/localcli/models.json` y describe los modelos descargados con su ruta, su repositorio de origen y los metadatos para identificarlos. El modelo seleccionado se guarda aquí (`seleccionado`) y en el estado de la aplicación, **nunca** en `localcli.json`. Un catálogo ausente o mal formado se trata como vacío, sin sobrescribirlo.

**La carpeta de cada modelo es su nombre (`id`).** Ese nombre —no la ruta del GGUF ni el nombre del archivo— es lo que se lista y selecciona en la interfaz y lo que se le nombra a `llama-server`; la `ruta` del GGUF es solo el detalle interno de arranque ([[backend/04-infrastructure/CONFIGURATION]] §8).

```json
{
  "modelos": [
    {
      "id": "qwen3-14b",
      "ruta": "~/.config/localcli/models/qwen3-14b/qwen3-14b-q4_k_m.gguf",
      "repo": "QuantFactory/Qwen3-14B-GGUF",
      "archivo": "qwen3-14b-q4_k_m.gguf",
      "ventana": 4096,
      "tamano": 9123456789
    }
  ],
  "seleccionado": "qwen3-14b"
}
```

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                              | Estado    | Archivos                       | Verificación                                                                 |
| --------- | ---------- | -------------------------------------------------------------------------------------------------- | --------- | ------------------------------ | ---------------------------------------------------------------------------- |
| T-B004-01 | crear      | Tipo del catálogo y lectura/escritura de `models.json`                                              | completada | `internal/models/catalogo.go`  | Round-trip del JSON conserva todos los campos                                 |
| T-B004-02 | crear      | Alta de un modelo al terminar su descarga (id, ruta, repo, archivo, ventana, tamaño)                | completada | `internal/models/catalogo.go`  | Tras descargar, el modelo aparece registrado con su ruta en `models/`         |
| T-B004-03 | actualizar | Ubicar el GGUF en `models/<modelo>/` —carpeta igual al nombre del modelo— y derivar la ruta canónica | completada | `internal/models/catalogo.go`  | `ruta` apunta al archivo real dentro de `models/<modelo>/`                    |
| T-B004-04 | crear      | Selección del modelo activo (`seleccionado`) y su persistencia                                      | completada | `internal/models/catalogo.go`  | Elegir un modelo lo deja como `seleccionado`; reiniciar lo conserva           |
| T-B004-05 | crear      | Comprobar que el GGUF existe antes de darlo por usable                                              | completada | `internal/models/catalogo.go`  | Un modelo catalogado con archivo borrado se marca como no usable             |
| T-B004-06 | crear      | Catálogo ausente o mal formado se trata como vacío, sin sobrescribir                                | completada | `internal/models/catalogo.go`  | Un `models.json` ilegible avisa y no se pisa                                  |
| T-B004-07 | actualizar | Exponer el nombre del modelo —su carpeta— y su ruta a los consumidores (T-B005 y la TUI)            | completada | `internal/models`              | El gestor de `llama-server` obtiene la ruta del modelo activo                 |
| T-B004-08 | actualizar | Derivar el nombre del modelo a partir de la carpeta de su ruta (`base(filepath.Dir(ruta))`)         | completada | `internal/models/nombre.go`    | El nombre coincide con la carpeta, no con el nombre del archivo GGUF          |

**Dependencias:** `01`→`02`→`03`,`04`; `04`→`05`; `03`→`08`; `08`→`07`; `01`→`06`.
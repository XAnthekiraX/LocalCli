# Test de Herramientas LocalCli

## 1. Leer archivos (`leer_archivo`)
**Descripción**: Leer archivos por rangos de línea.
**Test**: 
- Leer archivo README.md (líneas 1-10)
- Leer archivo este.md (líneas 1-100)

## 2. Listar carpetas (`listar_carpeta`)
**Descripción**: Mostrar contenido de una carpeta.
**Test**:
- Listar carpeta . (nivel raíz)
- Listar carpeta tests

## 3. Buscar archivos (`buscar_archivos`)
**Descripción**: Buscar archivos por nombre.
**Test**:
- Buscar "test" en nombres de archivos
- Buscar "README" en nombres de archivos

## 4. Buscar en contenido (`buscar_en_archivos`)
**Descripción**: Buscar texto dentro de archivos.
**Test**:
- Buscar "LocalCli" en todo el proyecto
- Buscar "herramientas" en carpeta tests

## 5. Crear archivos (`crear_archivo`)
**Descripción**: Crear archivos nuevos.
**Test**:
- Crear archivo `tests/herramienta_test_1.txt` con contenido "Prueba 1"

## 6. Escribir archivos (`escribir_archivo`)
**Descripción**: Sobrescribir archivos completos.
**Test**:
- Escribir archivo `tests/test_ejemplo.txt` con contenido "Contenido nuevo de prueba"

## 7. Editar archivos (`editar_archivo`)
**Descripción**: Aplicar sustituciones parciales.
**Test**:
- Editar archivo `tests/test_ejemplo.txt`: buscar "Contenido nuevo" reemplazar con "Contenido actualizado"

## 8. Eliminar archivos (`eliminar_archivo`)
**Descripción**: Borrar archivos.
**Test**:
- Eliminar archivo `tests/herramienta_test_1.txt`

## 9. Crear carpetas (`crear_carpeta`)
**Descripción**: Crear carpetas.
**Test**:
- Crear carpeta `tests/subcarpeta_1`

## 10. Eliminar carpetas (`eliminar_carpeta`)
**Descripción**: Borrar carpetas.
**Test**:
- Eliminar carpeta `tests/subcarpeta_1` (después de crearla)

## 11. Ejecutar comandos (`ejecutar_comando`)
**Descripción**: Ejecutar comandos de terminal.
**Test**:
- Ejecutar comando `pwd` (mostrar directorio actual)

---
*Nota: Las herramientas relacionadas con internet (buscar_en_internet y abrir_pagina) no se incluyen ya que no están implementadas.*
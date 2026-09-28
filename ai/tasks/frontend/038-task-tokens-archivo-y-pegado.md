> T-F038 — tokens de archivo y de pegado en la entrada: cualquier archivo pegado o arrastrado se muestra como `[nombre.ext]` (no solo las imágenes), y un texto de varias líneas como `[PEGADO N líneas]`; si todas las líneas son rutas de archivos, va un token por archivo. Al enviar se expanden al valor real.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §adjuntos y criterios de aceptación.
- [[specs/SPEC-OLLAMA-PERFIL]] — las imágenes viajan en base64 (eso no cambia).
- [[frontend/01-domain/DOMAIN]] — ficha `input` y reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §5: estados de espera y la entrada.
- [[frontend/05-quality/TESTING]] — qué se prueba de la entrada.

## Contexto

`adjuntos.go` (T-F026) tokenizaba solo rutas de **imagen**; cualquier otro archivo pegado o arrastrado quedaba como ruta cruda, y un pegado de varias líneas se insertaba entero. Esta tarea generaliza el mecanismo: `adjuntos` reconoce **cualquier archivo** existente y resume el **pegado multilínea**. El envío de imágenes en base64 no cambia: `RutasImagen`/`AdjuntosDe` siguen restringidos a imágenes.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F038-01 | actualizar | `adjuntos`: reconocer cualquier ruta (`RutasExistentes`/`esRuta`), no solo imágenes, al tokenizar; deduplicar por nombre conservando la ruta del segundo | completada | `internal/tui/adjuntos.go` | Test `TestAdjuntosAnotaUnArchivoNoImagen`, `TestAdjuntosDejaLaRutaInexistenteTalCual` |
| T-F038-02 | actualizar | `adjuntos`: `Anotar` con camino multilínea —`[PEGADO N líneas]` y un token por archivo si todas las líneas son archivos— y tokens de pegado únicos | completada | `internal/tui/adjuntos.go` | Tests `TestAdjuntosPegadoMultilineaMuestraElToken`, `TestAdjuntosUnSaltoFinalNoEsPegado`, `TestAdjuntosVariosArchivosUnTokenCadaUno`, `TestAdjuntosDosPegadosIgualesNoSePisan` |
| T-F038-03 | actualizar | Tests de componente: pegar un archivo muestra el token y envía la ruta; pegar multilínea resume y envía el texto entero; la bienvenida resume igual | completada | `internal/tui/adjuntos_test.go` | `TestPegarUnArchivoMuestraElTokenYEnviaLaRuta`, `TestPegarTextoMultilineaMuestraElTokenYEnviaElTexto`, `TestPegarMultilineaEnLaBienvenidaMuestraElToken` |
| T-F038-04 | actualizar | Documentar los tokens en `SPEC-INTERFAZ`, `DOMAIN`, `INTERFACES`, `TESTING` y `FRONTEND` | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/05-quality/TESTING.md`, `ai/docs/frontend/FRONTEND.md` | Documentación coherente con el código |
| T-F038-05 | actualizar | `adjuntos`: reconocer carpetas (`esCarpeta`) y mostrarlas como `[CARPETA N elementos]` (entradas de primer nivel); cuentan como ruta en el pegado multilínea | completada | `internal/tui/adjuntos.go` | Tests `TestAdjuntosAnotaUnaCarpeta`, `TestAdjuntosCarpetaConteoEnSingularYPlural`, `TestAdjuntosMultilineaConCarpeta`, `TestAdjuntosDosCarpetasIgualesNoSePisan` |

Dependencias: T-F038-01 y T-F038-02 antes de T-F038-03; T-F038-05 sobre T-F038-02; T-F038-04 revisa todas.

## Notas

- Las rutas valen **archivos o carpetas**: un archivo es `[nombre.ext]` y una carpeta `[CARPETA N elementos]` (entradas de primer nivel; una carpeta ilegible conserva su ruta).
- Las rutas se separan por espacios: una con espacios en el nombre no se tokeniza (igual que las imágenes hoy).
- No cambia `app.go` ni `welcome.go`: los dos puntos de pegado ya llaman a `Anotar`.
- El envío de imágenes en base64 queda igual; un archivo o carpeta que no es imagen viaja solo como texto (su ruta).

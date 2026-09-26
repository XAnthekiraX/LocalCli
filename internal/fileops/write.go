package fileops

// write.go — T-B008-05: aplicar escritura y edición de archivo solo tras
// aprobación, registrando el antes y el después.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §4
// ("toda escritura pasa por aprobación, incluso dentro de un flujo en curso"),
// ai/docs/backend/05-quality/VALIDATION.md §1 ("`crear_archivo` falla si el
// archivo ya existe; `escribir_archivo` sobrescribe; borrar pide confirmación
// explícita") y ai/docs/database/02-rules/DATA_FLOW.md §6 (escritura y registro
// son un solo paso lógico).
//
// Decisión fijada para `editar_archivo`: el campo `cambio` es un reemplazo
// exacto de un texto. Lleva el texto a buscar, una línea separadora `---` y el
// reemplazo. El texto buscado tiene que aparecer exactamente una vez; si no,
// la edición es ambigua y no se aplica. TOOLS-DTO no define otro formato, así
// que este es el mínimo que se puede aplicar sin adivinar.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"localcli/internal/store"
	"localcli/internal/tools"
)

// SeparadorEdicion divide el campo `cambio` de `editar_archivo`: lo anterior es
// el texto exacto a buscar y lo posterior su reemplazo.
const SeparadorEdicion = "\n---\n"

// partirEdicion interpreta el campo `cambio`.
func partirEdicion(cambio string) (buscar, reemplazar string, err error) {
	partes := strings.SplitN(cambio, SeparadorEdicion, 2)
	if len(partes) != 2 {
		return "", "", nuevoError(CodigoArgumentosInvalidos,
			"la edición debe llevar el texto a buscar y su reemplazo separados por una línea `---`")
	}
	if partes[0] == "" {
		return "", "", nuevoError(CodigoArgumentosInvalidos,
			"la edición no puede buscar un texto vacío")
	}
	return partes[0], partes[1], nil
}

// CrearArchivo crea un archivo nuevo. Falla si ya existe (E_PATH_EXISTS): no
// sobrescribe, para no destruir algo por accidente cuando se pretendía crear.
func (o *Ops) CrearArchivo(ctx context.Context, ruta, contenido string) (tools.RespuestaEscritura, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if _, statErr := os.Stat(abs); statErr == nil {
		return tools.RespuestaEscritura{}, nuevoError(CodigoRutaExiste,
			"el archivo "+ruta+" ya existe; usa escribir_archivo para sobrescribirlo")
	}
	if err := o.aprobar(ctx, store.OpCrearArchivo, ruta, false); err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo preparar la carpeta de %s: %w", ruta, err)
	}
	if err := os.WriteFile(abs, []byte(contenido), 0o644); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo crear %s: %w", ruta, err)
	}
	if err := o.registrar(store.OpCrearArchivo, mustRelativa(o.Proyecto, abs), "", contenido); err != nil {
		// Revertir: el archivo no existía antes.
		_ = os.Remove(abs)
		return tools.RespuestaEscritura{}, err
	}
	return tools.RespuestaEscritura{Confirmacion: "archivo creado", Ruta: ruta}, nil
}

// EscribirArchivo sobrescribe el contenido entero. Antes de aprobar ya se sabe
// qué contenido va a quedar, que es sobre lo que decide la persona.
func (o *Ops) EscribirArchivo(ctx context.Context, ruta, contenido string) (tools.RespuestaEscritura, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEscritura{}, err
	}
	before := ""
	existia := false
	if datos, rErr := os.ReadFile(abs); rErr == nil {
		before = string(datos)
		existia = true
	}
	if err := o.aprobar(ctx, store.OpEscribirArchivo, ruta, false); err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo preparar la carpeta de %s: %w", ruta, err)
	}
	if err := os.WriteFile(abs, []byte(contenido), 0o644); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo escribir %s: %w", ruta, err)
	}
	if err := o.registrar(store.OpEscribirArchivo, mustRelativa(o.Proyecto, abs), before, contenido); err != nil {
		revertirArchivo(abs, before, existia)
		return tools.RespuestaEscritura{}, err
	}
	return tools.RespuestaEscritura{Confirmacion: "archivo escrito", Ruta: ruta}, nil
}

// EditarArchivo aplica una edición parcial (reemplazo exacto de un texto).
func (o *Ops) EditarArchivo(ctx context.Context, ruta, cambio string) (tools.RespuestaEscritura, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEscritura{}, err
	}
	datos, err := os.ReadFile(abs)
	if err != nil {
		return tools.RespuestaEscritura{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo leer "+ruta+" para editarlo: "+err.Error())
	}
	before := string(datos)
	buscar, reemplazar, err := partirEdicion(cambio)
	if err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if n := strings.Count(before, buscar); n != 1 {
		return tools.RespuestaEscritura{}, nuevoError(CodigoArgumentosInvalidos,
			"el texto a editar aparece "+strconv.Itoa(n)+" veces en "+ruta+"; la edición debe ser inequívoca")
	}
	after := strings.Replace(before, buscar, reemplazar, 1)

	if err := o.aprobar(ctx, store.OpEditarArchivo, ruta, false); err != nil {
		return tools.RespuestaEscritura{}, err
	}
	if err := os.WriteFile(abs, []byte(after), 0o644); err != nil {
		return tools.RespuestaEscritura{}, fmt.Errorf("no se pudo editar %s: %w", ruta, err)
	}
	if err := o.registrar(store.OpEditarArchivo, mustRelativa(o.Proyecto, abs), before, after); err != nil {
		revertirArchivo(abs, before, true)
		return tools.RespuestaEscritura{}, err
	}
	return tools.RespuestaEscritura{Confirmacion: "archivo editado", Ruta: ruta}, nil
}

// EliminarArchivo borra un archivo tras confirmación explícita. El antes se
// guarda para poder revertir y auditar; el después es nulo porque ya no existe.
func (o *Ops) EliminarArchivo(ctx context.Context, ruta string) (tools.RespuestaEliminar, error) {
	abs, err := Resolver(o.Proyecto, ruta)
	if err != nil {
		return tools.RespuestaEliminar{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return tools.RespuestaEliminar{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo borrar "+ruta+": "+err.Error())
	}
	if info.IsDir() {
		return tools.RespuestaEliminar{}, nuevoError(CodigoArgumentosInvalidos,
			ruta+" es una carpeta; usa eliminar_carpeta")
	}
	datos, _ := os.ReadFile(abs)
	before := string(datos)

	if err := o.aprobar(ctx, store.OpEliminarArchivo, ruta, true); err != nil {
		return tools.RespuestaEliminar{}, err
	}
	if err := os.Remove(abs); err != nil {
		return tools.RespuestaEliminar{}, fmt.Errorf("no se pudo borrar %s: %w", ruta, err)
	}
	if err := o.registrar(store.OpEliminarArchivo, mustRelativa(o.Proyecto, abs), before, ""); err != nil {
		// Revertir: recrear el archivo tal como estaba.
		_ = os.WriteFile(abs, []byte(before), 0o644)
		return tools.RespuestaEliminar{}, err
	}
	return tools.RespuestaEliminar{Confirmacion: "archivo borrado"}, nil
}

// revertirArchivo deja el archivo como estaba antes del cambio.
func revertirArchivo(abs, before string, existia bool) {
	if existia {
		_ = os.WriteFile(abs, []byte(before), 0o644)
		return
	}
	_ = os.Remove(abs)
}

// mustRelativa es la ruta del historial. La resolución de la frontera ya
// garantiza que está dentro del proyecto, así que un fallo aquí no es
// recuperable por el llamador: se deja la ruta absoluta como último recurso.
func mustRelativa(proyecto, abs string) string {
	rel, err := Relativa(proyecto, abs)
	if err != nil {
		return abs
	}
	return rel
}

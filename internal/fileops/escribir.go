package fileops

import (
	"context"
	"os"
	"path/filepath"

	"localcli/internal/store"
	"localcli/internal/tools"
)

// EscribirHandlers devuelve los handlers de las herramientas de escritura.
func (o *Operaciones) EscribirHandlers() map[string]tools.Ejecutar {
	return map[string]tools.Ejecutar{
		"crear_archivo":    o.crearArchivo,
		"escribir_archivo": o.escribirArchivo,
		"editar_archivo":   o.editarArchivo,
		"eliminar_archivo": o.eliminarArchivo,
		"crear_carpeta":    o.crearCarpeta,
		"eliminar_carpeta": o.eliminarCarpeta,
	}
}

func (o *Operaciones) crearArchivo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionCrearArchivo)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, "", c, "Crear archivo", false)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	if _, err := os.Stat(abs); err == nil {
		return corregible(tools.CodPathExists + ": el archivo ya existe: " + Canonica(o.Raiz, abs))
	}
	if err := os.WriteFile(abs, []byte(p.Contenido), 0o644); err != nil {
		return corregible("no se pudo crear el archivo: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "crear_archivo", Canonica(o.Raiz, abs), nil, &p.Contenido)
	return okEscritura("archivo creado: " + Canonica(o.Raiz, abs)), nil
}

func (o *Operaciones) escribirArchivo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionEscribirArchivo)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, "", c, "Sobrescribir archivo", false)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	antes := leerExistente(abs)
	if err := os.WriteFile(abs, []byte(p.Contenido), 0o644); err != nil {
		return corregible("no se pudo escribir el archivo: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "escribir_archivo", Canonica(o.Raiz, abs), &antes, &p.Contenido)
	return okEscritura("archivo escrito: " + Canonica(o.Raiz, abs)), nil
}

func (o *Operaciones) editarArchivo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionEditarArchivo)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, "", c, "Editar archivo", false)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	datos, err := os.ReadFile(abs)
	if err != nil {
		return corregible("no se pudo leer el archivo: " + Canonica(o.Raiz, abs))
	}
	antes := string(datos)
	despues, err := aplicarReemplazos(antes, p.Reemplazos)
	if err != nil {
		return corregible(err.Error() + " [" + Canonica(o.Raiz, abs) + "]")
	}
	if despues == antes {
		return corregible("los reemplazos no cambiaron el archivo: " + Canonica(o.Raiz, abs))
	}
	if err := os.WriteFile(abs, []byte(despues), 0o644); err != nil {
		return corregible("no se pudo editar el archivo: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "editar_archivo", Canonica(o.Raiz, abs), &antes, &despues)
	return okEscritura("archivo editado: " + Canonica(o.Raiz, abs)), nil
}

func (o *Operaciones) eliminarArchivo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionEliminarArchivo)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, p.Motivo, c, "Borrar archivo", true)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	antes := leerExistente(abs)
	if err := os.Remove(abs); err != nil {
		return corregible("no se pudo borrar el archivo: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "eliminar_archivo", Canonica(o.Raiz, abs), &antes, nil)
	return okEscritura("archivo borrado: " + Canonica(o.Raiz, abs)), nil
}

func (o *Operaciones) crearCarpeta(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionCrearCarpeta)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, p.Motivo, c, "Crear carpeta", false)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	if _, err := os.Stat(abs); err == nil {
		return corregible(tools.CodPathExists + ": la carpeta ya existe: " + Canonica(o.Raiz, abs))
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return corregible("no se pudo crear la carpeta: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "crear_carpeta", Canonica(o.Raiz, abs), nil, nil)
	return okEscritura("carpeta creada: " + Canonica(o.Raiz, abs)), nil
}

func (o *Operaciones) eliminarCarpeta(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	p, ok := args.(tools.PeticionEliminarCarpeta)
	if !ok {
		return corregible(tools.CodBadArgs + ": argumentos inesperados")
	}
	abs, res, err := o.preparar(ctx, p.Ruta, p.Motivo, c, "Borrar carpeta", true)
	if err != nil || res != nil {
		return devolver(res, err)
	}
	if abs == filepath.Clean(o.Raiz) {
		return corregible(tools.CodBadArgs + ": no se borra la carpeta del proyecto")
	}
	if err := os.RemoveAll(abs); err != nil {
		return corregible("no se pudo borrar la carpeta: " + Canonica(o.Raiz, abs))
	}
	o.registrar(c.SesionID, "eliminar_carpeta", Canonica(o.Raiz, abs), nil, nil)
	return okEscritura("carpeta borrada: " + Canonica(o.Raiz, abs)), nil
}

// preparar resuelve la ruta, valida la frontera y pide aprobación. Devuelve la
// ruta absoluta, o un resultado ya compuesto si hay que cortar.
func (o *Operaciones) preparar(ctx context.Context, ruta, motivo string, c tools.Contexto, accion string, peligrosa bool) (string, *tools.Resultado, error) {
	abs, codigo := Resolver(o.Raiz, ruta)
	if codigo != "" {
		return "", &tools.Resultado{Error: codigo}, nil
	}
	if err := ValidarAcceso(ctx, o.Raiz, abs, motivo, c); err != nil {
		return "", nil, err
	}
	solicitud := tools.Solicitud{Titulo: accion, Descripcion: Canonica(o.Raiz, abs), Motivo: motivo, Peligrosa: peligrosa}
	if err := pedirPermiso(ctx, c, solicitud); err != nil {
		return "", nil, err
	}
	return abs, nil, nil
}

func devolver(res *tools.Resultado, err error) (tools.Resultado, error) {
	if err != nil {
		return resultado(err)
	}
	if res != nil {
		return *res, nil
	}
	return tools.Resultado{}, nil
}

func okEscritura(mensaje string) tools.Resultado {
	return tools.Resultado{Salida: mensaje}
}

func leerExistente(abs string) string {
	datos, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	return string(datos)
}

// registrar guarda la escritura en `change_history`; un fallo no revierte el
// cambio, pero se reporta.
func (o *Operaciones) registrar(sesionID, operacion, ruta string, antes, despues *string) {
	if o.DB == nil {
		return
	}
	_ = store.RegistrarCambio(o.DB, sesionID, operacion, ruta, antes, despues)
}

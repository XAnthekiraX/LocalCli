package fileops

// list.go — T-B008-03: listar y buscar dentro de la frontera.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §2 y §3
// (`listar_carpeta` devuelve las entradas de un nivel; `buscar_archivos` las
// rutas que coinciden; `buscar_en_archivos` las coincidencias con archivo y
// línea). Todas las rutas que salen son relativas a la carpeta del proyecto,
// como las que entran.

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"localcli/internal/tools"
)

// MaxCoincidencias acota cuántas coincidencias se devuelven. No lo fija la
// documentación; es defensivo: una búsqueda por contenido no puede llenar el
// contexto con miles de líneas.
const MaxCoincidencias = 200

// ListarCarpeta devuelve los nombres de las entradas de un nivel. Es el camino
// de quien lista sin poder pedir permiso: una ruta de fuera no se toca.
func ListarCarpeta(proyecto, ruta string) (tools.RespuestaListarCarpeta, error) {
	return (&Ops{Proyecto: proyecto}).ListarCarpeta(context.Background(), ruta)
}

// ListarCarpeta devuelve los nombres de las entradas de un nivel. Si la ruta
// sale de la carpeta del proyecto, pide permiso antes de listarla.
func (o *Ops) ListarCarpeta(ctx context.Context, ruta string) (tools.RespuestaListarCarpeta, error) {
	abs, err := o.rutaDeLectura(ctx, "listar_carpeta", ruta)
	if err != nil {
		return tools.RespuestaListarCarpeta{}, err
	}
	entradas, err := os.ReadDir(abs)
	if err != nil {
		return tools.RespuestaListarCarpeta{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo listar "+ruta+": "+err.Error())
	}
	// `os.ReadDir` ya devuelve ordenadas por nombre, que es el orden estable
	// que espera el contrato.
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		nombres = append(nombres, e.Name())
	}
	return tools.RespuestaListarCarpeta{Entradas: nombres}, nil
}

// BuscarArchivos devuelve las rutas (relativas) cuyo nombre de archivo coincide
// con el patrón. El patrón es de `filepath.Match`.
func BuscarArchivos(proyecto, patron string) (tools.RespuestaBuscarArchivos, error) {
	return (&Ops{Proyecto: proyecto}).BuscarArchivos(context.Background(), patron)
}

// BuscarArchivos recorre el proyecto entero: la búsqueda siempre empieza dentro
// de la carpeta abierta, así que no hay permiso que pedir.
func (o *Ops) BuscarArchivos(ctx context.Context, patron string) (tools.RespuestaBuscarArchivos, error) {
	raiz, _, err := Resolver(o.Proyecto, ".")
	if err != nil {
		return tools.RespuestaBuscarArchivos{}, err
	}
	if strings.TrimSpace(patron) == "" {
		return tools.RespuestaBuscarArchivos{}, nuevoError(CodigoArgumentosInvalidos, "falta el patrón")
	}
	var rutas []string
	err = filepath.WalkDir(raiz, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // un subárbol ilegible no aborta la búsqueda entera
		}
		if d.IsDir() {
			if abs != raiz && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		ok, mErr := filepath.Match(patron, d.Name())
		if mErr == nil && ok {
			if rel, rErr := Relativa(o.Proyecto, abs); rErr == nil {
				rutas = append(rutas, rel)
			}
		}
		return nil
	})
	if err != nil {
		return tools.RespuestaBuscarArchivos{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo buscar: "+err.Error())
	}
	sort.Strings(rutas)
	return tools.RespuestaBuscarArchivos{Rutas: rutas}, nil
}

// BuscarEnArchivos busca el patrón en el contenido. Sin `ruta`, recorre el
// proyecto entero; con `ruta`, solo esa subcarpeta o archivo.
func BuscarEnArchivos(proyecto, patron, ruta string) (tools.RespuestaBuscarEnArchivos, error) {
	return (&Ops{Proyecto: proyecto}).BuscarEnArchivos(context.Background(), patron, ruta)
}

// BuscarEnArchivos busca el patrón en el contenido. Si la ruta sale de la
// carpeta del proyecto, pide permiso antes de recorrerla.
func (o *Ops) BuscarEnArchivos(ctx context.Context, patron, ruta string) (tools.RespuestaBuscarEnArchivos, error) {
	if strings.TrimSpace(patron) == "" {
		return tools.RespuestaBuscarEnArchivos{}, nuevoError(CodigoArgumentosInvalidos, "falta el patrón")
	}
	re, err := regexp.Compile(patron)
	if err != nil {
		return tools.RespuestaBuscarEnArchivos{}, nuevoError(CodigoArgumentosInvalidos,
			"el patrón no es válido: "+err.Error())
	}
	raiz, _, err := Resolver(o.Proyecto, ".")
	if err != nil {
		return tools.RespuestaBuscarEnArchivos{}, err
	}
	objetivo := raiz
	if strings.TrimSpace(ruta) != "" {
		if objetivo, err = o.rutaDeLectura(ctx, "buscar_en_archivos", ruta); err != nil {
			return tools.RespuestaBuscarEnArchivos{}, err
		}
	}

	var out []tools.Coincidencia
	recorrer := func(abs string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(out) >= MaxCoincidencias {
			return filepath.SkipAll
		}
		coincidencias, cErr := coincidenciasEnArchivo(o.Proyecto, abs, re, MaxCoincidencias-len(out))
		if cErr == nil {
			out = append(out, coincidencias...)
		}
		return nil
	}
	if info, statErr := os.Stat(objetivo); statErr == nil && !info.IsDir() {
		coincidencias, cErr := coincidenciasEnArchivo(o.Proyecto, objetivo, re, MaxCoincidencias)
		if cErr == nil {
			out = append(out, coincidencias...)
		}
	} else {
		_ = filepath.WalkDir(objetivo, recorrer)
	}
	if out == nil {
		out = []tools.Coincidencia{}
	}
	return tools.RespuestaBuscarEnArchivos{Coincidencias: out}, nil
}

// coincidenciasEnArchivo lee un archivo línea a línea y devuelve las que casan.
// Un archivo binario o ilegible se salta sin abortar la búsqueda.
func coincidenciasEnArchivo(proyecto, abs string, re *regexp.Regexp, cupo int) ([]tools.Coincidencia, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rel, err := Relativa(proyecto, abs)
	if err != nil {
		return nil, err
	}
	var out []tools.Coincidencia
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	linea := 0
	for sc.Scan() {
		linea++
		texto := sc.Text()
		if !re.MatchString(texto) {
			continue
		}
		out = append(out, tools.Coincidencia{
			Archivo:   rel,
			Linea:     linea,
			Fragmento: strings.TrimSpace(texto),
		})
		if len(out) >= cupo {
			break
		}
	}
	return out, nil
}

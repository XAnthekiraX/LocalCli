package docs

// loader.go — T-B003-01: recorrer ai/docs/**/*.md y cargar el contenido.
//
// La documentación del proyecto es la fuente de verdad (DOMAIN.md §"En
// archivos"): no se guarda en SQLite, se lee entera en memoria al arrancar y
// sobre ella se construye el grafo (DECISIONS.md: "grafo en memoria,
// reconstruido al arrancar"). Este paquete solo lee; nunca escribe.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Doc es un documento markdown cargado: su ruta relativa a la raíz de docs,
// el contenido en crudo, el frontmatter parseado y los wiki-links del cuerpo.
type Doc struct {
	Path     string   // relativo a la raíz, con barra inclinada: "backend/DECISIONS.md"
	Raw      []byte   // contenido completo del archivo
	Front    Meta     // frontmatter YAML (vacío si no tiene)
	Body     string   // cuerpo sin el bloque de frontmatter
	Links    []string // [[destino]] del cuerpo, en orden de aparición
	HasFront bool     // true si el archivo declaraba frontmatter
}

// CargarDocs recorre recursivamente la carpeta dada (típicamente
// <proyecto>/ai/docs), carga cada .md y parsea frontmatter y enlaces.
// Un error de parseo localizado (frontmatter inválido) se acumula en
// loadErrs y NO aborta la carga (T-B003-06); solo un fallo de lectura o de
// recorrido interrumpe. El resultado va ordenado por ruta para que el grafo
// sea determinista.
func CargarDocs(raiz string) (map[string]*Doc, []error, error) {
	info, err := os.Stat(raiz)
	if err != nil || !info.IsDir() {
		// No es "documento no encontrado": la raíz entera no es documentación,
		// así que la etapa no puede seguir. E_STAGE_FAILED, con el motivo.
		return nil, nil, NuevoErrStage(fmt.Sprintf("%s no es una carpeta de documentación", raiz))
	}
	var (
		docs     = map[string]*Doc{}
		loadErrs []error
	)
	walkErr := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(raiz, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(p)
		if err != nil {
			// un archivo ilegible se reporta localizado y la carga sigue.
			loadErrs = append(loadErrs, fmt.Errorf("%s: E_DOC_PARSE: %w", rel, err))
			return nil
		}
		doc := &Doc{Path: rel, Raw: raw}
		frente, cuerpo, tuvo, ferr := ParseFrontmatter(raw)
		if ferr != nil {
			// YAML inválido: error localizado con el código documentado,
			// documento descartado, carga intacta (T-B003-06).
			loadErrs = append(loadErrs, NuevoErrParse(rel, ferr))
			return nil
		}
		doc.Front, doc.Body, doc.HasFront = frente, cuerpo, tuvo
		doc.Links = ExtraerEnlaces(cuerpo)
		docs[rel] = doc
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
		return nil, loadErrs, walkErr
	}
	return docs, loadErrs, nil
}

// Rutas devuelve las rutas de los documentos cargados, ordenadas.
func Rutas(docs map[string]*Doc) []string {
	out := make([]string, 0, len(docs))
	for r := range docs {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

package docs

// documento.go — T-B020-01: el documento cargado y su frontmatter.
//
// La documentación es fuente de verdad (DOMAIN.md §"En archivos"): no se guarda
// en SQLite, se lee entera en memoria al arrancar y sobre ella se construye el
// grafo (DECISIONS.md: "grafo en memoria, reconstruido al arrancar"). Este
// paquete solo lee; nunca escribe.

// Meta es el frontmatter parseado. Tags recoge las claves convencionales de
// etiquetado (tags/etiquetas/keywords); el resto queda en Claves y Listas.
type Meta struct {
	Tags   []string
	Claves map[string]string
	Listas map[string][]string
}

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

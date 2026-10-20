package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Entrada es un modelo descargado registrado en el catálogo. `ID` es su nombre
// (el de su carpeta en `models/`); `Ruta` es el GGUF real, detalle de arranque.
type Entrada struct {
	ID      string `json:"id"`
	Ruta    string `json:"ruta"`
	Repo    string `json:"repo"`
	Archivo string `json:"archivo"`
	// MMProj es el nombre del proyector multimodal dentro de la carpeta del
	// modelo, si se registró. Vacío: se busca por convención (ver `Proyector`).
	MMProj  string `json:"mmproj,omitempty"`
	Ventana int    `json:"ventana"`
	Tamano  int64  `json:"tamano"`
}

// RutaReal devuelve la ruta del GGUF con `~` resuelto al directorio del usuario.
func (e Entrada) RutaReal() string { return ExpandirRuta(e.Ruta) }

// Directorio es la carpeta del modelo: la que contiene su GGUF.
func (e Entrada) Directorio() string { return filepath.Dir(e.RutaReal()) }

// Proyector devuelve la ruta del proyector multimodal del modelo: la registrada
// o, si no, la primera `mmproj*.gguf` de su carpeta. Vacío si no hay ninguno.
func (e Entrada) Proyector() string {
	if strings.TrimSpace(e.MMProj) != "" {
		return filepath.Join(e.Directorio(), e.MMProj)
	}
	coincidencias, err := filepath.Glob(filepath.Join(e.Directorio(), "mmproj*.gguf"))
	if err != nil || len(coincidencias) == 0 {
		return ""
	}
	return coincidencias[0]
}

// TieneMMProj indica si el modelo trae proyector multimodal.
func (e Entrada) TieneMMProj() bool { return e.Proyector() != "" }

// Usable indica si el GGUF del modelo sigue existiendo en disco.
func (e Entrada) Usable() bool {
	if strings.TrimSpace(e.Ruta) == "" {
		return false
	}
	_, err := os.Stat(e.RutaReal())
	return err == nil
}

// ContextoReal devuelve el contexto del GGUF del modelo: el registrado al darlo
// de alta o, si la entrada es antigua y no lo trae, el leído al vuelo del propio
// archivo. 0 cuando no se puede determinar.
func (e Entrada) ContextoReal() int {
	if e.Ventana > 0 {
		return e.Ventana
	}
	return contextoDeArchivo(e)
}

// contextoDeArchivo lee el contexto del GGUF si está en disco; 0 si no se puede.
func contextoDeArchivo(e Entrada) int {
	if strings.TrimSpace(e.Ruta) == "" {
		return 0
	}
	ctx, err := ContextoDeGGUF(e.RutaReal())
	if err != nil {
		return 0
	}
	return ctx
}

// Catalogo es el registro de modelos descargados y el modelo activo.
type Catalogo struct {
	Modelos      []Entrada `json:"modelos"`
	Seleccionado string    `json:"seleccionado"`
}

// Gestor administra el catálogo persistido en `ruta` (`models.json`).
type Gestor struct {
	ruta string
}

// NuevoGestor devuelve un gestor sobre el `models.json` indicado.
func NuevoGestor(ruta string) *Gestor { return &Gestor{ruta: ruta} }

// DirectorioModels es la carpeta de modelos que acompaña al catálogo.
func (g *Gestor) DirectorioModels() string {
	return filepath.Join(filepath.Dir(g.ruta), "models")
}

// RutaCanonica deriva la ruta del GGUF de un modelo: `models/<id>/<archivo>`.
func (g *Gestor) RutaCanonica(id, archivo string) string {
	return filepath.Join(g.DirectorioModels(), id, archivo)
}

// Cargar lee el catálogo. Un archivo ausente o vacío devuelve un catálogo vacío;
// uno mal formado devuelve un catálogo vacío y un error, y nunca se sobrescribe
// por sí solo.
func (g *Gestor) Cargar() (*Catalogo, error) { return LeerCatalogo(g.ruta) }

// LeerCatalogo lee el catálogo del archivo indicado.
func LeerCatalogo(ruta string) (*Catalogo, error) {
	datos, err := os.ReadFile(ruta)
	if os.IsNotExist(err) {
		return &Catalogo{}, nil
	}
	if err != nil {
		return &Catalogo{}, errorf("no se pudo leer %s: %w", ruta, err)
	}
	if strings.TrimSpace(string(datos)) == "" {
		return &Catalogo{}, nil
	}
	var c Catalogo
	if err := json.Unmarshal(datos, &c); err != nil {
		return &Catalogo{}, errorf("%s está mal formado: %w", ruta, err)
	}
	return &c, nil
}

// Guardar escribe el catálogo de forma atómica (temporal + renombrado).
func (c *Catalogo) Guardar(ruta string) error {
	datos, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errorf("no se pudo serializar el catálogo: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return errorf("no se pudo crear %s: %w", filepath.Dir(ruta), err)
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o644); err != nil {
		return errorf("no se pudo escribir %s: %w", ruta, err)
	}
	if err := os.Rename(tmp, ruta); err != nil {
		return errorf("no se pudo colocar %s: %w", ruta, err)
	}
	return nil
}

// Alta registra o reemplaza un modelo por su id. No escribe si el catálogo
// existente está mal formado: evita pisarlo. Si la entrada no trae ventana, se
// lee del GGUF: es el contexto real del modelo, no un valor de arranque.
func (g *Gestor) Alta(e Entrada) error {
	e.ID = strings.TrimSpace(e.ID)
	if !nombreValido(e.ID) {
		return errorf("nombre de modelo inválido: %q", e.ID)
	}
	if motivo := MotivoCabezaDraft(e.Ruta); motivo != "" {
		return errorf("no se registra %q: %s", e.ID, motivo)
	}
	if e.Ventana <= 0 {
		e.Ventana = contextoDeArchivo(e)
	}
	c, err := g.Cargar()
	if err != nil {
		return err
	}
	c.poner(e)
	return c.Guardar(g.ruta)
}

// AltaDescarga registra el modelo recién descargado con su ruta real y, opcionalmente,
// el nombre del archivo mmproj (proyector multimodal) dentro de la misma carpeta.
func (g *Gestor) AltaDescarga(id string, v Variante, ruta string, ventana int, mmproj string) error {
	return g.Alta(Entrada{
		ID:      id,
		Ruta:    ruta,
		Repo:    v.Repo,
		Archivo: v.Archivo,
		MMProj:  mmproj,
		Ventana: ventana,
		Tamano:  v.Tamano,
	})
}

// Seleccionar deja `id` como modelo activo. El modelo debe estar en el catálogo.
func (g *Gestor) Seleccionar(id string) error {
	c, err := g.Cargar()
	if err != nil {
		return err
	}
	if _, ok := c.Buscar(id); !ok {
		return errorf("no hay un modelo %q en el catálogo", id)
	}
	c.Seleccionado = id
	return c.Guardar(g.ruta)
}

// Activo devuelve el modelo seleccionado.
func (c *Catalogo) Activo() (Entrada, bool) {
	if c.Seleccionado == "" {
		return Entrada{}, false
	}
	return c.Buscar(c.Seleccionado)
}

// Buscar devuelve la entrada con ese id.
func (c *Catalogo) Buscar(id string) (Entrada, bool) {
	for _, e := range c.Modelos {
		if e.ID == id {
			return e, true
		}
	}
	return Entrada{}, false
}

// poner agrega la entrada o reemplaza la existente con el mismo id.
func (c *Catalogo) poner(e Entrada) {
	for i := range c.Modelos {
		if c.Modelos[i].ID == e.ID {
			c.Modelos[i] = e
			return
		}
	}
	c.Modelos = append(c.Modelos, e)
}

// ExpandirRuta resuelve un `~` inicial al directorio del usuario.
func ExpandirRuta(ruta string) string {
	if ruta != "~" && !strings.HasPrefix(ruta, "~/") {
		return ruta
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ruta
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(ruta, "~"), "/"))
}

// nombreValido indica si el id es un nombre de carpeta plano y no una ruta.
func nombreValido(id string) bool {
	return id != "" && id != "." && id != ".." && filepath.Base(id) == id
}

// keys.go — T-F001/T-F010-04: la persistencia del mapa de teclas.
//
// Fuente de verdad: FRONTEND.md §3 ("el mapa de teclas vive en
// ~/.config/localcli/keys.json, fuera del proyecto, porque es preferencia del
// usuario y no contenido del proyecto") e INTERFACES §4 (el atajo se reasigna y
// "el cambio se guarda sin reiniciar la aplicación").
//
// El archivo guarda el mapa completo acción → tecla. Al cargar se parte de los
// atajos de fábrica y se aplican los ajustes del usuario, de modo que un mapa
// incompleto nunca deja una acción sin tecla. Lo que no valide (acción
// desconocida, tecla vacía, duplicados) se rechaza: quien carga decide si
// caer a los valores de fábrica.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// nombreDeAccion da el nombre estable de cada acción en el JSON. Es parte del
// formato del archivo: cambiarlo rompe la configuración del usuario.
func nombreDeAccion(a Accion) string {
	switch a {
	case AccionEnviar:
		return "enviar"
	case AccionSalir:
		return "salir"
	case AccionPanel:
		return "panel"
	case AccionSelector:
		return "selector"
	case AccionRazonamiento:
		return "razonamiento"
	case AccionAprobaciones:
		return "aprobaciones"
	case AccionAprobar:
		return "aprobar"
	case AccionDeclinar:
		return "declinar"
	case AccionPausar:
		return "pausar"
	case AccionCancelar:
		return "cancelar"
	case AccionCerrarSelector:
		return "cerrar_selector"
	case AccionAyuda:
		return "ayuda"
	}
	return ""
}

// accionDeNombre resuelve el nombre del JSON a acción.
func accionDeNombre(s string) (Accion, bool) {
	for _, a := range []Accion{
		AccionEnviar, AccionSalir, AccionPanel, AccionSelector, AccionRazonamiento,
		AccionAprobaciones, AccionAprobar, AccionDeclinar, AccionPausar,
		AccionCancelar, AccionCerrarSelector, AccionAyuda,
	} {
		if nombreDeAccion(a) == s {
			return a, true
		}
	}
	return AccionNinguna, false
}

// descripcionAccion da la ayuda de una acción que no viene en el mapa de
// fábrica (por ejemplo, `pausar`, que existe pero no trae tecla por defecto).
var descripcionAccion = map[Accion]string{
	AccionPausar:         "pausar la cola en curso",
	AccionAprobar:        "aprobar la propuesta seleccionada",
	AccionDeclinar:       "declinar la propuesta seleccionada",
	AccionCerrarSelector: "cerrar el selector",
}

// archivoKeys es la forma del JSON en disco.
type archivoKeys struct {
	Atajos map[string]string `json:"atajos"`
}

// RutaKeys resuelve la ruta del archivo de preferencias fuera del proyecto:
// ~/.config/localcli/keys.json (FRONTEND.md §3).
func RutaKeys() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("tui: no se pudo resolver la carpeta del usuario: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "keys.json"), nil
}

// CargarKeys carga el mapa desde la ruta del usuario.
func CargarKeys() ([]Atajo, error) {
	ruta, err := RutaKeys()
	if err != nil {
		return nil, err
	}
	return CargarKeysDesde(filepath.Dir(ruta))
}

// CargarKeysDesde carga el mapa desde un directorio. Sin archivo no hay error:
// se devuelven los atajos de fábrica (T-F001-04, archivo ausente tolerado).
func CargarKeysDesde(dir string) ([]Atajo, error) {
	bruto, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if os.IsNotExist(err) {
		return AtajosPorDefecto(), nil
	}
	if err != nil {
		return nil, err
	}
	var archivo archivoKeys
	if err := json.Unmarshal(bruto, &archivo); err != nil {
		return nil, err
	}

	// Se parte de los de fábrica y se aplican los ajustes del usuario.
	atajos := AtajosPorDefecto()
	puestos := map[Accion]bool{}
	for i := range atajos {
		if tecla, ok := archivo.Atajos[nombreDeAccion(atajos[i].Accion)]; ok {
			atajos[i].Tecla = tecla
			puestos[atajos[i].Accion] = true
		}
	}
	// El resto de entradas son acciones sin tecla de fábrica (p. ej. pausar)
	// que el usuario asignó. Una acción desconocida es un mapa roto: se
	// rechaza entero, no se adivina.
	for nombre, tecla := range archivo.Atajos {
		accion, ok := accionDeNombre(nombre)
		if !ok {
			return nil, fmt.Errorf("tui: keys.json nombra una acción desconocida: %s", nombre)
		}
		if puestos[accion] {
			continue
		}
		atajos = append(atajos, Atajo{
			Tecla:       tecla,
			Accion:      accion,
			Descripcion: descripcionAccion[accion],
		})
	}

	// Duplicados, teclas vacías y acciones sin sentido se rechazan al cargar,
	// igual que al guardar.
	if err := ValidarAtajos(atajos); err != nil {
		return nil, err
	}
	return atajos, nil
}

// GuardarKeys guarda el mapa en la ruta del usuario.
func GuardarKeys(atajos []Atajo) error {
	ruta, err := RutaKeys()
	if err != nil {
		return err
	}
	return GuardarKeysEn(filepath.Dir(ruta), atajos)
}

// GuardarKeysEn guarda el mapa en un directorio, creándolo si no existe. Un
// mapa inválido (dos acciones con la misma tecla, tecla vacía) se rechaza antes
// de escribir nada (T-F001-03: el duplicado se rechaza al guardar).
func GuardarKeysEn(dir string, atajos []Atajo) error {
	if err := ValidarAtajos(atajos); err != nil {
		return err
	}
	archivo := archivoKeys{Atajos: map[string]string{}}
	for _, a := range atajos {
		archivo.Atajos[nombreDeAccion(a.Accion)] = a.Tecla
	}
	bruto, err := json.MarshalIndent(archivo, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "keys.json"), bruto, 0o644)
}

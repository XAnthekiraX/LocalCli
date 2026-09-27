// keys.go — T-F001/T-F010-04 actualizado por T-F012-01: la persistencia del
// mapa de teclas.
//
// Fuente de verdad: specs/SPEC-KEYBINDS.md (§Configuración: keys.json con
// `leader`, `leader_timeout_ms` y `keybinds` con listas; lo ausente se rellena
// con los valores de fábrica y lo inválido se rechaza al cargar) y
// FRONTEND.md §3 ("el mapa de teclas vive en ~/.config/localcli/keys.json,
// fuera del proyecto, porque es preferencia del usuario y no contenido del
// proyecto") e INTERFACES §4 (el atajo se reasigna y "el cambio se guarda sin
// reiniciar la aplicación").
//
// La migración mantiene la compatibilidad: un keys.json del esquema antiguo
// (`atajos` planos, action→tecla) produce el mapa nuevo equivalente, y guardar
// escribe siempre el esquema nuevo. Serializar y deserializar es idempotente.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// --- el archivo de configuración (T-F012-01) -----------------------------------
//
// Las constantes y nombres estables son parte del formato en disco: cambiarlos
// rompe la configuración del usuario.

// nombreDeAccionLegacy es el identificador que usaba el keys.json anterior a
// T-F012 (`{"atajos": {"panel": "ctrl+d"}}`). Al cargar, cada nombre antiguo se
// traduce al ID nuevo equivalente; así un archivo viejo produce el mapa nuevo
// sin que el usuario lo toque.
func nombreDeAccionLegacy(a Accion) string {
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

// accionDeNombre resuelve un identificador —antiguo o nuevo— a acción. La lista
// es abierta: añadir una acción añade su ID aquí y en nombreDeAccion.
func accionDeNombre(s string) (Accion, bool) {
	for _, a := range OrdenDeAcciones() {
		if nombreDeAccion(a) == s || nombreDeAccionLegacy(a) == s {
			return a, true
		}
	}
	return AccionNinguna, false
}

// normalizarNombreLegacy tolera la errata del esquema antiguo: los archivos
// keys.json previos a T-F012 escribieron «cerrar selector» (con espacio) para
// la acción dismiss. Al cargar se normaliza al identificador con guion bajo,
// que es el nombre legacy que reconoce accionDeNombre.
func normalizarNombreLegacy(s string) string {
	if s == "cerrar selector" {
		return "cerrar_selector"
	}
	return s
}

// nombreDeAccion es el identificador estable de cada acción (tabla de
// SPEC-KEYBINDS §Acción). Es también la clave con la que viaja en el JSON.
func nombreDeAccion(a Accion) string {
	switch a {
	case AccionEnviar:
		return "send"
	case AccionSalir:
		return "app_exit"
	case AccionPanel:
		return "panel_toggle"
	case AccionSelector:
		return "session_picker"
	case AccionSesionNueva:
		return "session_new"
	case AccionEliminarSesion:
		return "session_delete"
	case AccionRazonamiento:
		return "reasoning_toggle"
	case AccionAprobaciones:
		return "approvals_toggle"
	case AccionAprobar:
		return "approve"
	case AccionDeclinar:
		return "decline"
	case AccionPausar:
		return "pause"
	case AccionCancelar:
		return "cancel"
	case AccionCerrarSelector:
		return "dismiss"
	case AccionAyuda:
		return "command_palette"
	case AccionModalModelos:
		return "model_picker"
	case AccionCiclarAgente:
		return "agent_cycle"
	case AccionSubir:
		return "up"
	case AccionBajar:
		return "down"
	case AccionChatSubir:
		return "chat_scroll_up"
	case AccionChatBajar:
		return "chat_scroll_down"
	case AccionChatPaginaArriba:
		return "chat_page_up"
	case AccionChatPaginaAbajo:
		return "chat_page_down"
	}
	return ""
}

// archivoKeys es la forma del JSON en disco. `keybinds` es el esquema nuevo
// (lista de literales por acción, líder y timeout configurables); `atajos` es
// el esquema antiguo, plano, que sigue cargándose por compatibilidad.
type archivoKeys struct {
	Líder     string              `json:"leader,omitempty"`
	TimeoutMs int                 `json:"leader_timeout_ms,omitempty"`
	Keybinds  map[string][]string `json:"keybinds,omitempty"`
	Atajos    map[string]string   `json:"atajos,omitempty"`
}

// accionesDeBienvenida son las únicas acciones que existen en la pantalla de
// bienvenida (SPEC-INTERFAZ §Pantalla de bienvenida: aquí solo se escribe, se
// envía con enter, se alterna el agente con Tab —el indicador se ve también
// aquí—, se abre el modal de modelos con Ctrl+X m y se sale; los modelos se
// eligen en el modal, no en la línea de entrada). El resto del mapa
// —panel, aprobaciones, modal de atajos— no debe resolverse mientras esta vista está
// activa; welcome.go filtra contra esta lista después de pasar por el resolver,
// sin tocar el estado del líder.
func accionesDeBienvenida() map[Accion]bool {
	return map[Accion]bool{
		AccionSalir:          true,
		AccionEnviar:         true,
		AccionModalModelos:   true,
		AccionSelector:       true,
		AccionEliminarSesion: true,
		AccionCerrarSelector: true,
		AccionSubir:          true,
		AccionBajar:          true,
		AccionCiclarAgente:   true,
	}
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

// CargarKeymap carga el mapa completo desde la ruta del usuario.
func CargarKeymap() (*Keymap, error) {
	ruta, err := RutaKeys()
	if err != nil {
		return nil, err
	}
	return CargarKeymapDesde(filepath.Dir(ruta))
}

// CargarKeymapDesde carga el mapa desde un directorio. Sin archivo no hay
// error: se devuelve el mapa de fábrica (T-F001-04, archivo ausente tolerado).
// Un keys.json antiguo (`atajos` planos) produce el mapa nuevo equivalente: lo
// ausente se rellena con los valores de fábrica y lo inválido (ID desconocido,
// literal vacío, duplicado, timeout fuera de rango) se rechaza entero, no se
// adivina (SPEC-KEYBINDS §Configuración).
func CargarKeymapDesde(dir string) (*Keymap, error) {
	fabrica := MapasPorDefecto()
	bruto, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if os.IsNotExist(err) {
		return NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, fabrica)
	}
	if err != nil {
		return nil, err
	}
	var archivo archivoKeys
	if err := json.Unmarshal(bruto, &archivo); err != nil {
		return nil, err
	}

	lider := archivo.Líder
	timeout := archivo.TimeoutMs
	porAccion := map[Accion][]string{}
	for accion, literales := range fabrica {
		porAccion[accion] = append([]string{}, literales...)
	}

	// Esquema nuevo: keybinds con listas. Una lista vacía deshabilita la acción.
	for nombre, literales := range archivo.Keybinds {
		accion, ok := accionDeNombre(normalizarNombreLegacy(nombre))
		if !ok {
			return nil, fmt.Errorf("tui: keys.json nombra una acción desconocida: %s", nombre)
		}
		lista := append([]string{}, literales...)
		if lista == nil {
			lista = []string{}
		}
		porAccion[accion] = lista
	}

	// Esquema antiguo: un literal plano por acción, compatible con el mapa
	// nuevo porque las acciones compartidas conservan su significado.
	for nombre, tecla := range archivo.Atajos {
		accion, ok := accionDeNombre(normalizarNombreLegacy(nombre))
		if !ok {
			return nil, fmt.Errorf("tui: keys.json nombra una acción desconocida: %s", nombre)
		}
		porAccion[accion] = []string{tecla}
	}

	return NuevoKeymap(lider, timeout, porAccion)
}

// CargarKeys carga solo los atajos del archivo del usuario (o los de fábrica si
// no carga), como hacía antes de T-F012. El resto del programa consume ya el
// Keymap; esta función queda para quien solo necesite la lista.
func CargarKeys() ([]Atajo, error) {
	km, err := CargarKeymap()
	if err != nil {
		return nil, err
	}
	return km.Entradas(), nil
}

// CargarKeysDesde carga la lista de atajos desde un directorio.
func CargarKeysDesde(dir string) ([]Atajo, error) {
	km, err := CargarKeymapDesde(dir)
	if err != nil {
		return nil, err
	}
	return km.Entradas(), nil
}

// GuardarKeymap guarda el mapa completo (líder, timeout y keybinds) en la ruta
// del usuario.
func GuardarKeymap(km *Keymap) error {
	ruta, err := RutaKeys()
	if err != nil {
		return err
	}
	return GuardarKeymapEn(filepath.Dir(ruta), km)
}

// GuardarKeymapEn guarda el mapa completo en un directorio, creándolo si no
// existe. Serializar y volver a cargar es idempotente (verificación de
// T-F012-01): lo que se escribe es exactamente lo que se lee.
func GuardarKeymapEn(dir string, km *Keymap) error {
	archivo := archivoKeys{
		Líder:     km.Lider(),
		TimeoutMs: int(km.TimeoutMs() / time.Millisecond),
		Keybinds:  map[string][]string{},
	}
	for _, e := range km.Entradas() {
		archivo.Keybinds[nombreDeAccion(e.Accion)] = literalesDe(e)
	}
	return escribirArchivo(dir, archivo)
}

// literalesDe devuelve los literales de una acción tal como van en el JSON: la
// secuencia con líder se guarda con su prefijo `<leader>`, no con la líder ya
// expandida, para que cambiar de líder en el archivo no rompa nada.
func literalesDe(e Atajo) []string {
	literales := []string{}
	for _, sec := range e.Secuencias {
		literales = append(literales, sec.Describir())
	}
	return literales
}

// GuardarKeys guarda el mapa en la ruta del usuario.
func GuardarKeys(atajos []Atajo) error {
	ruta, err := RutaKeys()
	if err != nil {
		return err
	}
	return GuardarKeysEn(filepath.Dir(ruta), atajos)
}

// GuardarKeysEn guarda la lista de atajos en un directorio, creándolo si no
// existe. Un mapa inválido (dos acciones con el mismo literal, literal vacío)
// se rechaza antes de escribir nada (T-F001-03: el duplicado se rechaza al
// guardar). Se escribe el esquema nuevo: leer y escribir este archivo produce
// el mismo mapa (idempotencia, T-F012-01).
func GuardarKeysEn(dir string, atajos []Atajo) error {
	if err := ValidarAtajos(atajos); err != nil {
		return err
	}
	archivo := archivoKeys{
		Líder:     LíderPorDefecto,
		TimeoutMs: TimeoutPorDefectoMs,
		Keybinds:  map[string][]string{},
	}
	for _, e := range atajos {
		archivo.Keybinds[nombreDeAccion(e.Accion)] = literalesDe(e)
	}
	return escribirArchivo(dir, archivo)
}

// escribirArchivo serializa y escribe keys.json con los permisos de siempre.
func escribirArchivo(dir string, archivo archivoKeys) error {
	bruto, err := json.MarshalIndent(archivo, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "keys.json"), bruto, 0o644)
}

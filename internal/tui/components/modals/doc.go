// Package modals guarda los tres modales de la TUI —modelos, sesiones y
// atajos—: listas centradas con un elemento resaltado, un aviso, un pie de
// teclas y un descarte. Comparten la mecánica de `Modal`: uno abierto a la vez,
// `↑`/`↓` navegan, `Enter` aplica y cierra, `Esc` descarta sin cambiar nada.
//
// Es solo estado y pintado: no leen del motor ni de la base, no conocen el mapa
// de teclas y no deciden nada. La lista llega ya compuesta, la elección se
// entrega a quien la abrió —el `app`— y quien abre y cierra es él: los modales
// no se abren entre ellos.
package modals

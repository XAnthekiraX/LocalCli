package modals

import (
	"strings"
	"testing"
)

func sesionesDePrueba() []Sesion {
	return []Sesion{
		{ID: "s1", Nombre: "primera", Estado: "inactiva"},
		{ID: "s2", Nombre: "segunda", Estado: EstadoTrabajando},
		{ID: "s3", Nombre: "tercera", Estado: "esperando_permiso"},
	}
}

// Al abrir, el resaltado arranca en la sesión activa y su fila va marcada.
func TestElResaltadoArrancaEnLaSesionActiva(t *testing.T) {
	var sm Sesiones
	sm.Abrir("s2")
	sm.Fijar(sesionesDePrueba(), nil)

	if sm.Indice != 1 {
		t.Fatalf("el resaltado quedó en %d, quiero 1", sm.Indice)
	}
	if !strings.Contains(sm.Lineas[1], "←") {
		t.Fatalf("la sesión activa debería ir marcada: %q", sm.Lineas[1])
	}
	if strings.Contains(sm.Lineas[0], "←") {
		t.Fatalf("solo la activa va marcada: %q", sm.Lineas[0])
	}
	elegida, ok := sm.SesionElegida()
	if !ok || elegida.ID != "s2" {
		t.Fatalf("la sesión elegida = %+v", elegida)
	}
}

// Cada fila lleva el nombre y el estado tal como llegan de la base.
func TestCadaFilaLlevaNombreYEstado(t *testing.T) {
	var sm Sesiones
	sm.Abrir("")
	sm.Fijar(sesionesDePrueba(), nil)

	plano := strings.Join(sm.Lineas, "\n")
	for _, quiero := range []string{"primera  [inactiva]", "segunda  [trabajando]", "tercera  [esperando_permiso]"} {
		if !strings.Contains(plano, quiero) {
			t.Errorf("falta la fila %q:\n%s", quiero, plano)
		}
	}
}

// Sin sesiones —o con una lectura fallida— queda el aviso, y una lista que llega
// tarde no reabre el modal.
func TestSinSesionesQuedaElAviso(t *testing.T) {
	var sm Sesiones
	sm.Abrir("")
	sm.Fijar(nil, nil)
	if sm.Aviso != AvisoSinSesiones {
		t.Fatalf("el aviso = %q", sm.Aviso)
	}

	var tarde Sesiones
	tarde.Abrir("")
	tarde.Cerrar()
	tarde.Fijar(sesionesDePrueba(), nil)
	if tarde.Abierto || len(tarde.Lista) != 0 {
		t.Fatalf("una lista tardía no debería rellenar nada: %+v", tarde)
	}
}

// Una sesión que no trabaja se elimina a la primera; una que trabaja pide
// confirmación y solo entrega la eliminación a la segunda pulsación.
func TestEliminarPideConfirmacionSoloSiTrabaja(t *testing.T) {
	var sm Sesiones
	sm.Abrir("")
	sm.Fijar(sesionesDePrueba(), nil)
	sm.Mover(1) // la que trabaja

	if _, adelante := sm.PedirEliminar(); adelante {
		t.Fatal("una sesión que trabaja no debería borrarse sin confirmar")
	}
	if !sm.Confirmando() {
		t.Fatal("debería quedar una confirmación pendiente")
	}
	if !strings.Contains(strings.Join(sm.Lineas, "\n"), "¿seguro que deseas eliminar?") {
		t.Fatalf("la pregunta debería verse:\n%s", strings.Join(sm.Lineas, "\n"))
	}
	sesion, adelante := sm.PedirEliminar()
	if !adelante || sesion.ID != "s2" {
		t.Fatalf("la segunda pulsación debería entregar la sesión: %+v %v", sesion, adelante)
	}
	if sm.Confirmando() {
		t.Fatal("la pregunta ya contestada no debería quedar armada")
	}

	// Cancelar la pregunta la retira sin borrar nada.
	sm.PedirEliminar()
	sm.CancelarConfirmacion()
	if sm.Confirmando() {
		t.Fatal("cancelar debería retirar la pregunta")
	}
	if strings.Contains(strings.Join(sm.Lineas, "\n"), "¿seguro que deseas eliminar?") {
		t.Fatal("la pregunta cancelada no debería seguir pintada")
	}
	if _, adelante := sm.PedirEliminar(); adelante {
		t.Fatal("tras cancelar, la eliminación debería volver a preguntar")
	}
}

// Una sesión inactiva se entrega a la primera y sin pregunta.
func TestUnaSesionInactivaSeEntregaSinPreguntar(t *testing.T) {
	var sm Sesiones
	sm.Abrir("")
	sm.Fijar(sesionesDePrueba(), nil)

	sesion, adelante := sm.PedirEliminar()
	if !adelante || sesion.ID != "s1" || sm.Confirmando() {
		t.Fatalf("una sesión inactiva debería entregarse directamente: %+v %v", sesion, adelante)
	}
}

// El estado y el título de una sesión se refrescan en la lista abierta sin mover
// el resaltado, y una sesión que no está en la lista no toca nada.
func TestLaListaSeRefrescaSinMoverElResaltado(t *testing.T) {
	var sm Sesiones
	sm.Abrir("s1")
	sm.Fijar(sesionesDePrueba(), nil)
	sm.Mover(2)

	sm.ActualizarEstado("s3", "terminada")
	sm.ActualizarNombre("s3", "ya no se llama así")

	if sm.Indice != 2 {
		t.Fatalf("el resaltado se movió solo: %d", sm.Indice)
	}
	if sm.Lista[2].Estado != "terminada" || sm.Lista[2].Nombre != "ya no se llama así" {
		t.Fatalf("la fila no se refrescó: %+v", sm.Lista[2])
	}
	sm.ActualizarEstado("ajena", "inactiva")
	sm.ActualizarNombre("ajena", "otra")
	if len(sm.Lista) != 3 {
		t.Fatalf("una sesión ajena no debería entrar en la lista: %d", len(sm.Lista))
	}
}

// El modal de sesiones se pinta centrado con su pie, que anuncia el borrado.
func TestElModalDeSesionesSePintaConSuPie(t *testing.T) {
	var sm Sesiones
	sm.Abrir("s1")
	sm.Fijar(sesionesDePrueba(), nil)

	plano := strings.Join(strings.Split(sm.Render(100, 20), "\n"), "\n")
	if !strings.Contains(plano, "SESIONES") || !strings.Contains(plano, PieSesiones) {
		t.Fatalf("el render del modal:\n%s", plano)
	}
}

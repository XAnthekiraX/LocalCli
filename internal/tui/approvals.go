// approvals.go — T-B014-06: el panel de aprobaciones.
//
// Fuente de verdad: SPEC-INTERFAZ-ATAJOS §Formato de una línea del panel
// (`nombre de la sesión | acción propuesta | aprobar | declinar | <opción por
// definir>`), §Flujos alternativos ("Dos sesiones esperan a la vez: aparecen
// como dos líneas independientes", "La sesión terminó mientras esperaba: la línea
// se marca como obsoleta") y §Reglas ("Resuelve una aprobación solo afecta a la
// sesión de esa línea", "El panel se puede abrir y cerrar sin detener el trabajo
// de ninguna sesión").
//
// Este panel no es el panel de datos (SPEC-INTERFAZ-ATAJOS: "El de datos es de
// lectura y solo información; este son decisiones que esperan tu respuesta"). Por
// eso vive en su propio archivo y su propio estado: compartir la disposición
// habría mezclado lo que se lee con lo que se decide.
package tui

import "strings"

// Aprobacion es una línea del panel: una decisión esperando respuesta.
type Aprobacion struct {
	ID          string
	Sesion      string
	Descripcion string
	// Obsoleta marca una aprobación que ya no aplica (la sesión terminó
	// mientras esperaba).
	Obsoleta bool
}

// Aprobaciones es el panel de decisiones pendientes.
type Aprobaciones struct {
	Abierto bool
	Items   []Aprobacion
	Indice  int
}

// Fijar reemplaza la lista, conservando la selección dentro de los límites.
//
// El panel se abre solo cuando hay algo que decidir y se cierra cuando no queda
// nada: es un panel de decisiones, y vacío no aporta nada a la pantalla.
func (a *Aprobaciones) Fijar(items []Aprobacion) {
	a.Items = items
	if a.Indice >= len(items) {
		a.Indice = len(items) - 1
	}
	if a.Indice < 0 {
		a.Indice = 0
	}
	a.Abierto = len(items) > 0
}

// Pendientes cuenta las que siguen esperando decisión.
func (a *Aprobaciones) Pendientes() int {
	n := 0
	for _, it := range a.Items {
		if !it.Obsoleta {
			n++
		}
	}
	return n
}

// Mover cambia la línea seleccionada.
func (a *Aprobaciones) Mover(delta int) {
	if len(a.Items) == 0 {
		return
	}
	a.Indice = (a.Indice + delta + len(a.Items)) % len(a.Items)
}

// Seleccionada devuelve la aprobación en el cursor.
func (a *Aprobaciones) Seleccionada() (Aprobacion, bool) {
	if len(a.Items) == 0 || a.Indice < 0 || a.Indice >= len(a.Items) {
		return Aprobacion{}, false
	}
	return a.Items[a.Indice], true
}

// Resolver saca una línea del panel: la decisión se fue a la sesión dueña y la
// línea ya no espera nada. Devuelve la aprobación resuelta para que quien la
// reciba sepa a quién afecta.
func (a *Aprobaciones) Resolver(id string) (Aprobacion, bool) {
	for i, it := range a.Items {
		if it.ID != id {
			continue
		}
		a.Items = append(a.Items[:i], a.Items[i+1:]...)
		if a.Indice >= len(a.Items) && a.Indice > 0 {
			a.Indice--
		}
		return it, true
	}
	return Aprobacion{}, false
}

// Lineas devuelve cada aprobación en el formato documentado, con la seleccionada
// marcada. Una aprobación obsoleta se marca como tal.
func (a *Aprobaciones) Lineas() []string {
	out := make([]string, 0, len(a.Items))
	for i, it := range a.Items {
		marca := "  "
		if i == a.Indice {
			marca = "› "
		}
		opciones := "aprobar | declinar"
		if it.Obsoleta {
			opciones = "obsoleta"
		}
		out = append(out, marca+it.Sesion+" | "+it.Descripcion+" | "+opciones)
	}
	return out
}

// Render pinta el panel completo.
func (a *Aprobaciones) Render() string {
	if !a.Abierto {
		return ""
	}
	var b strings.Builder
	b.WriteString(estiloTitulo.Render("APROBACIONES") + "\n")
	lineas := a.Lineas()
	if len(lineas) == 0 {
		b.WriteString(estiloSistema.Render("(nada esperando decisión)") + "\n")
		return strings.TrimRight(b.String(), "\n")
	}
	for i, l := range lineas {
		if a.Items[i].Obsoleta {
			b.WriteString(estiloSistema.Render(l) + "\n")
			continue
		}
		if i == a.Indice {
			l = estiloUsuario.Render(l)
		}
		b.WriteString(l + "\n")
	}
	b.WriteString(estiloSistema.Render("(ctrl+a aprobar · ctrl+d declinar)"))
	return strings.TrimRight(b.String(), "\n")
}

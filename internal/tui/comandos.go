package tui

// comandos.go — la paleta de comandos de flujo.
//
// Fuente de verdad: [[specs/SPEC-INTERFAZ]] §Reglas de negocio ("Un flujo solo
// arranca con un comando explícito escrito en la entrada: /planificar, /crear,
// /actualizar, /eliminar, /resolver o /ejecutar") y [[specs/SPEC-MOTOR-FLUJOS]]
// §Reglas de negocio ("Un flujo no arranca solo: lo solicita el usuario con un
// comando explícito").
//
// Escribir `/` despliega la lista encima del input; las flechas la recorren, Tab
// autocompleta el comando resaltado —dejando la línea lista para escribir la
// petición detrás— y Enter lo ejecuta. El catálogo que se ofrece sale del puerto
// (los flujos oficiales más los que declara `ai/flows/*.json`, [[specs/SPEC-FLUJO-PERSONALIZADO]]);
// sin lista, cae al respaldo oficial de este paquete.
//
// El respaldo se repite aquí a propósito: `tui` no puede importar `flow`
// (tests/arquitectura_test.go §TestLimitesDeImporteEntreModulos), que es quien
// arma el catálogo real.

import "strings"

// ComandoFlujo es un flujo —oficial o propio— que se arranca escribiendo su
// comando.
type ComandoFlujo struct {
	// Nombre es el comando, con su barra inicial (por ejemplo "/resolver").
	Nombre string
	// Descripcion dice, en minúscula, para qué es el flujo; se pinta junto al
	// comando en la paleta.
	Descripcion string
}

// comandosDeFlujo es el catálogo OFICIAL de respaldo: los seis comandos que fija
// SPEC-INTERFAZ §Reglas de negocio. Lo usa la vista cuando el puerto no ofrece
// un catálogo del proyecto.
func comandosDeFlujo() []ComandoFlujo {
	return []ComandoFlujo{
		{Nombre: "/planificar", Descripcion: "planificar el proyecto desde cero"},
		{Nombre: "/crear", Descripcion: "crear una funcionalidad nueva"},
		{Nombre: "/actualizar", Descripcion: "actualizar una funcionalidad existente"},
		{Nombre: "/eliminar", Descripcion: "eliminar una funcionalidad"},
		{Nombre: "/resolver", Descripcion: "resolver un problema"},
		{Nombre: "/ejecutar", Descripcion: "ejecutar la cola de tareas"},
	}
}

// ComandoFlujoDe reconoce un comando de flujo al inicio de la línea dentro de la
// lista dada. Devuelve el comando y true solo si su nombre está en la lista;
// cualquier otro texto —incluido otro `/…`— es chat.
func ComandoFlujoDe(comandos []ComandoFlujo, texto string) (ComandoFlujo, bool) {
	campos := strings.Fields(strings.TrimSpace(texto))
	if len(campos) == 0 {
		return ComandoFlujo{}, false
	}
	for _, c := range comandos {
		if campos[0] == c.Nombre {
			return c, true
		}
	}
	return ComandoFlujo{}, false
}

// lineaDeComando compone la línea que se muestra en el chat al ejecutar un
// comando: el nombre canónico y, si la había, la petición que lo acompañaba.
// Ejecutado desde la paleta (texto parcial) la petición no existe todavía, así
// que la línea es solo el comando.
func lineaDeComando(c ComandoFlujo, escrito string) string {
	t := strings.TrimSpace(escrito)
	campos := strings.Fields(t)
	if len(campos) <= 1 {
		return c.Nombre
	}
	resto := strings.TrimSpace(strings.TrimPrefix(t, campos[0]))
	if resto == "" {
		return c.Nombre
	}
	return c.Nombre + " " + resto
}

// Paleta es la lista de comandos que se despliega encima del input mientras se
// escribe un comando. Guarda lo que hay que listar, el elemento resaltado y el
// texto con el que se calculó, para no rehacer el filtro en cada pintado.
type Paleta struct {
	// Abierto dice si hay lista que mostrar.
	Abierto  bool
	visibles []ComandoFlujo
	indice   int
	texto    string
	// comandos es el catálogo que se ofrece: los flujos del proyecto más
	// `/ejecutar`. Lo fija la vista al construirse; el filtro solo lo recorre.
	comandos []ComandoFlujo
}

// FijarComandos da a la paleta el catálogo que ofrece. Una lista vacía deja el
// respaldo oficial: la paleta nunca se queda sin comandos que ofrecer.
func (p *Paleta) FijarComandos(comandos []ComandoFlujo) {
	if len(comandos) == 0 {
		comandos = comandosDeFlujo()
	}
	p.comandos = comandos
}

// Filtrar recalcula la lista a partir de lo escrito y reinicia el resaltado
// cuando el texto cambia. Solo hay comandos que ofrecer si la línea empieza por
// `/` y todavía no tiene espacios: en cuanto se escribe la petición, la paleta
// se retira (el comando ya está elegido).
func (p *Paleta) Filtrar(texto string) {
	if texto == p.texto {
		return
	}
	p.texto = texto
	p.visibles = comandosQueEmpiezanPor(texto, p.comandos)
	p.indice = 0
	p.Abierto = len(p.visibles) > 0
}

// comandosQueEmpiezanPor devuelve los comandos cuyo nombre empieza por la línea
// escrita. Sin barra inicial o con espacios no hay comandos que ofrecer: en
// cuanto aparece un espacio empieza la petición y el comando ya está elegido.
func comandosQueEmpiezanPor(texto string, comandos []ComandoFlujo) []ComandoFlujo {
	if len(comandos) == 0 {
		comandos = comandosDeFlujo()
	}
	if !strings.HasPrefix(texto, "/") || strings.ContainsAny(texto, " \t\n") {
		return nil
	}
	var out []ComandoFlujo
	for _, c := range comandos {
		if strings.HasPrefix(c.Nombre, texto) {
			out = append(out, c)
		}
	}
	return out
}

// Mover cambia el resaltado sin salirse de la lista.
func (p *Paleta) Mover(delta int) {
	if len(p.visibles) == 0 {
		return
	}
	p.indice = (p.indice + delta + len(p.visibles)) % len(p.visibles)
}

// Seleccionado devuelve el comando resaltado.
func (p *Paleta) Seleccionado() (ComandoFlujo, bool) {
	if len(p.visibles) == 0 {
		return ComandoFlujo{}, false
	}
	return p.visibles[p.indice], true
}

// Render pinta la lista con el comando resaltado, encima del input. Devuelve ""
// si no hay nada que mostrar.
func (p *Paleta) Render() string {
	if !p.Abierto || len(p.visibles) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range p.visibles {
		fila := c.Nombre + "  " + c.Descripcion
		if i == p.indice {
			b.WriteString(estiloUsuario.Render("› "+fila) + "\n")
			continue
		}
		b.WriteString(estiloSistema.Render("  "+fila) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

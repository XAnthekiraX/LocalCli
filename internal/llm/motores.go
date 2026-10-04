// motores.go — T-B037-03: el registro de motores declarados.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/CONFIGURATION.md §8
// (estructura de `motores.json` y sus campos), ai/docs/specs/SPEC-MODELO-MOTOR
// (§El catálogo es cerrado, §Gestión de motores, §Flujo principal),
// ai/docs/backend/DECISIONS.md (registro global en archivo; editar exige
// reiniciar; una cola por motor) y ai/docs/backend/05-quality/ERRORS.md §3
// (E_MOTOR_TIPO_DESCONOCIDO y E_REGISTRO_MOTORES_INVALIDO).
//
// El registro vive en `~/.config/localcli/motores.json`: es una característica
// de la máquina —qué runtimes hay y dónde—, no del proyecto (DECISIONS). Este
// paquete guarda los motores declarados y expone sus operaciones; no decide
// qué motor usa cada sesión ni construye adaptadores (eso es T-B037-04).
//
// Ninguna operación de carga escribe el archivo del usuario: los defaults y
// los tipos desconocidos solo se resuelven en memoria, salvo que algo agregue,
// edite o elimine después.
package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// Los dos tipos del catálogo cerrado. Añadir un runtime es código, no
// configuración (SPEC-MODELO-MOTOR §El catálogo es cerrado).
const (
	TipoOllama   = "ollama"
	TipoLlamaCPP = "llamacpp"
)

// Direcciones por defecto de cada tipo (SPEC-MODELO-MOTOR §El catálogo es
// cerrado).
const (
	URLPorDefectoOllama   = "http://localhost:11434"
	URLPorDefectoLlamaCPP = "http://localhost:8080"
)

// EsTipoValido dice si `tipo` es uno de los dos tipos cerrados del catálogo.
func EsTipoValido(tipo string) bool {
	return tipo == TipoOllama || tipo == TipoLlamaCPP
}

// ExtensionesDeTipo devuelve las extensiones nativas que un tipo del catálogo
// cerrado admite (SPEC-MODELO-MOTOR §Núcleo común y extensiones nativas):
// `num_ctx`, `show` y `tags` en `ollama`; `props` en `llamacpp`. Un tipo
// desconocido no admite ninguna. Devuelve una lista nueva cada vez para que el
// llamante no pueda mutar la del catálogo.
func ExtensionesDeTipo(tipo string) []string {
	switch tipo {
	case TipoOllama:
		return []string{"num_ctx", "show", "tags"}
	case TipoLlamaCPP:
		return []string{"props"}
	default:
		return nil
	}
}

// Entrada es una fila declarada de `motores.json`: la configuración de un
// motor, no su adaptador (CONFIGURATION.md §8).
type Entrada struct {
	// ID es la identidad estable con la que las sesiones nombran al motor.
	// No cambia al editar nombre, tipo o URL, para no dejar sesiones huérfanas.
	ID string `json:"id"`
	// Nombre es la etiqueta visible: la del modal y la de la línea de estado.
	// Es la de la instancia, no la del tipo (dos `ollama` se distinguen así).
	Nombre string `json:"nombre"`
	// Tipo es el runtime: `ollama` o `llamacpp`. Es cerrado; determina el
	// adaptador que hablará con el servidor.
	Tipo string `json:"tipo"`
	// URL es la dirección base del servidor. Es del motor, no del tipo, así
	// que admite varias instancias del mismo tipo con direcciones distintas.
	URL string `json:"url"`
	// Activo dice si el motor se puede usar. Desactivar no borra nada: la
	// entrada sigue visible con sus modelos no disponibles.
	Activo bool `json:"activo"`
	// Extensiones son las extensiones nativas que esta instancia declara
	// (`num_ctx`, `show` y `tags` en `ollama`; `props` en `llamacpp`). Ausente o
	// vacío significa «solo núcleo común»: lo que la extensión aportaría queda
	// en desconocido, nunca en «no lo tiene» (SPEC-MODELO-MOTOR §Núcleo común y
	// extensiones nativas). El núcleo ignora cualquier extensión que no conozca.
	Extensiones []string `json:"extensiones,omitempty"`
}

// archivoMotores es la forma exacta de `~/.config/localcli/motores.json`
// (CONFIGURATION.md §8).
type archivoMotores struct {
	Motores []Entrada `json:"motores"`
	// ModelosPorMotor es la caché del último catálogo visto por motor: sirve
	// para que la lista abra al instante y para que un motor desactivado
	// siga mostrando qué tenía. No es una verdad: al conectar se pregunta al
	// motor y lo que responda manda (CONFIGURATION.md §8).
	ModelosPorMotor map[string][]string `json:"modelos_por_motor,omitempty"`
}

// Aviso es una advertencia de carga del registro, con su código documentado en
// ERRORS.md §3 y un mensaje para la persona. No corta nada: el registro sigue
// cargado con lo que pudo cargar.
type Aviso struct {
	Codigo  string
	Mensaje string
}

// Error implementa `error` para que los avisos se puedan imprimir y comparar
// igual que cualquier otro error.
func (a Aviso) Error() string { return a.Codigo + ": " + a.Mensaje }

// Registro es el conjunto de motores declarados, con su caché de modelos. El
// cero no es utilizable: cargar con Cargar.
type Registro struct {
	mu       sync.Mutex
	ruta     string
	entradas []Entrada
	modelos  map[string][]string
	// caches reúne lo que se cachea POR MOTOR: la ventana que fija el servidor,
	// la que declara cada modelo y su ficha de capacidades. Es por motor y no
	// por nombre de modelo porque dos motores del mismo tipo pueden declarar
	// cosas distintas del mismo nombre de modelo (SPEC-MODELO-MOTOR §Lo que
	// cambia por motor). No se persiste: el archivo guarda solo `motores.json`
	// con `modelos_por_motor`.
	caches map[string]*cacheMotor
	// adaptadores es el adaptador VIVO de cada motor, construido por su tipo
	// cerrado y cacheado por `id`: `llm` es el único sitio que instancia
	// adaptadores (BACKEND.md §3). Agregar, desactivar, reactivar y eliminar
	// lo construyen o lo destruyen al momento; editar NO lo toca.
	adaptadores map[string]Motor
	// pendientesDeReinicio marca los motores editados con adaptador vivo: el
	// cambio se aplica al reiniciar, no a la sesión en curso (DECISIONS).
	pendientesDeReinicio map[string]bool
}

// cacheMotor es la caché viva de un motor registrado.
type cacheMotor struct {
	// ventanaServidor es la ventana que fija el servidor cuando la lee el
	// harness (llama.cpp); 0 con ventanaLeida = el motor la declara por
	// petición (Ollama) y no la fija él.
	ventanaServidor int
	ventanaLeida    bool
	// contextos cachea, por modelo, la ventana que declara el modelo. Un 0
	// cacheado significa «el modelo no la declara» y se guarda igual para no
	// volver a listar.
	contextos map[string]int
	// capacidades cachea, por modelo, la ficha normalizada. Una ficha no se
	// cachea vacía: un fallo de red no debe fijar «no puede».
	capacidades map[string][]string
}

func nuevoCacheMotor() *cacheMotor {
	return &cacheMotor{contextos: map[string]int{}, capacidades: map[string][]string{}}
}

// RutaMotores resuelve la ruta del registro global del usuario, en la misma
// carpeta que config.json y keys.json (CONFIGURATION.md §8).
func RutaMotores() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("llm: no se pudo resolver la carpeta del usuario: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "motores.json"), nil
}

// porDefecto son los motores con los que se arranca sin registro: `ollama` y
// `llamacpp` con sus direcciones por defecto. Un registro ausente o ilegible
// nunca deja al usuario sin motor (SPEC-MODELO-MOTOR, criterios de aceptación).
func porDefecto() []Entrada {
	return []Entrada{
		{ID: "ollama-local", Nombre: "Ollama local", Tipo: TipoOllama, URL: URLPorDefectoOllama, Activo: true},
		{ID: "llamacpp-local", Nombre: "llama.cpp local", Tipo: TipoLlamaCPP, URL: URLPorDefectoLlamaCPP, Activo: true},
	}
}

// Cargar lee el registro desde `ruta` y lo deja en r, devolviendo los avisos
// de la carga (vacío si todo fue limpio):
//
//   - Archivo ausente: los dos motores por defecto, sin aviso.
//   - Archivo ilegible o mal formado: aviso E_REGISTRO_MOTORES_INVALIDO y los
//     dos motores por defecto (ERRORS.md §3: se avisa y se sigue, no se cae).
//   - Entrada con tipo desconocido: aviso E_MOTOR_TIPO_DESCONOCIDO nombrando
//     la entrada y el tipo, y se salta esa entrada; si no queda ninguna, se
//     usa el `ollama` por defecto (CONFIGURATION.md §8, ERRORS.md §3).
func (r *Registro) Cargar(ruta string) []Aviso {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruta = ruta
	r.modelos = map[string][]string{}
	r.caches = map[string]*cacheMotor{}
	r.adaptadores = map[string]Motor{}
	r.pendientesDeReinicio = map[string]bool{}

	bruto, err := os.ReadFile(ruta)
	if err != nil && !os.IsNotExist(err) {
		r.entradas = porDefecto()
		return []Aviso{{
			Codigo:  CodigoRegistroMotoresInvalido,
			Mensaje: "no se pudo leer motores.json (" + err.Error() + "); se usan los motores por defecto",
		}}
	}
	if os.IsNotExist(err) {
		r.entradas = porDefecto()
		return nil
	}

	var arch archivoMotores
	if err := json.Unmarshal(bruto, &arch); err != nil {
		r.entradas = porDefecto()
		return []Aviso{{
			Codigo:  CodigoRegistroMotoresInvalido,
			Mensaje: "motores.json está mal formado (" + err.Error() + "); se usan los motores por defecto",
		}}
	}

	var avisos []Aviso
	validas := make([]Entrada, 0, len(arch.Motores))
	for _, e := range arch.Motores {
		if !EsTipoValido(e.Tipo) {
			avisos = append(avisos, Aviso{
				Codigo: CodigoMotorTipoDesconocido,
				Mensaje: fmt.Sprintf("la entrada %q declara el tipo %q, que no es un motor conocido (solo %s y %s); se ignora",
					nombrarEntrada(e), e.Tipo, TipoOllama, TipoLlamaCPP),
			})
			continue
		}
		validas = append(validas, e)
	}
	// Un registro que no queda con ninguna entrada no deja al usuario sin
	// motor: se usa el `ollama` por defecto (ERRORS.md §3).
	if len(validas) == 0 {
		validas = porDefecto()[:1]
	}
	r.entradas = validas

	for id, ms := range arch.ModelosPorMotor {
		r.modelos[id] = append([]string(nil), ms...)
	}
	return avisos
}

// Guardar escribe el registro a la ruta de la última Cargar, creando la
// carpeta si no existe.
func (r *Registro) Guardar() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.guardar()
}

// guardar escribe sin tomar el candado (el llamante lo tiene). Serializar y
// volver a leer es idempotente.
func (r *Registro) guardar() error {
	if r.ruta == "" {
		return errors.New("llm: el registro no tiene ruta; cárgalo con Cargar antes de guardarlo")
	}
	bruto, err := json.MarshalIndent(archivoMotores{Motores: r.entradas, ModelosPorMotor: r.modelos}, "", "  ")
	if err != nil {
		return fmt.Errorf("llm: no se pudo serializar el registro: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o755); err != nil {
		return fmt.Errorf("llm: no se pudo crear la carpeta del registro: %w", err)
	}
	if err := os.WriteFile(r.ruta, bruto, 0o644); err != nil {
		return fmt.Errorf("llm: no se pudo escribir motores.json: %w", err)
	}
	return nil
}

// Agregar da de alta un motor y guarda. El `id` llega o se genera (es la
// identidad estable con la que las sesiones lo nombran); un `tipo` fuera del
// catálogo cerrado se rechaza con E_MOTOR_TIPO_DESCONOCIDO, y un `id` ya
// registrado también: dos motores no pueden compartir identidad.
func (r *Registro) Agregar(e Entrada) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !EsTipoValido(e.Tipo) {
		return &ErrorMotor{
			Codigo:   CodigoMotorTipoDesconocido,
			Mensaje:  fmt.Sprintf("el tipo %q no es un motor conocido (solo %s y %s)", e.Tipo, TipoOllama, TipoLlamaCPP),
			Detalle:  "tipo fuera del catálogo cerrado",
			sentinel: ErrMotorTipoDesconocido,
		}
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if r.indice(e.ID) >= 0 {
		return fmt.Errorf("llm: ya hay un motor registrado con id %q", e.ID)
	}
	// Agregar se aplica al momento: si el motor entra activo, su adaptador
	// queda construido y listo para el primer turno. Si no se puede construir,
	// no se registra a medias.
	if e.Activo {
		m, err := AdaptadorDeTipo(e.Tipo, e.URL)
		if err != nil {
			return err
		}
		r.iniciarMapas()
		r.adaptadores[e.ID] = m
	}
	r.entradas = append(r.entradas, e)
	delete(r.pendientesDeReinicio, e.ID)
	return r.guardar()
}

// Editar cambia nombre, tipo, URL o extensiones de un motor registrado y
// guarda. El `id` y
// el estado `activo` no cambian: la identidad no se reescribe y activar es
// Desactivar/Reactivar. El cambio se aplica al comportamiento al reiniciar
// (DECISIONS: el adaptador vivo no se toca); esta función solo actualiza el
// registro.
func (r *Registro) Editar(e Entrada) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !EsTipoValido(e.Tipo) {
		return &ErrorMotor{
			Codigo:   CodigoMotorTipoDesconocido,
			Mensaje:  fmt.Sprintf("el tipo %q no es un motor conocido (solo %s y %s)", e.Tipo, TipoOllama, TipoLlamaCPP),
			Detalle:  "tipo fuera del catálogo cerrado",
			sentinel: ErrMotorTipoDesconocido,
		}
	}
	i := r.indice(e.ID)
	if i < 0 {
		return fmt.Errorf("llm: no hay ningún motor registrado con id %q", e.ID)
	}
	r.entradas[i].Nombre = e.Nombre
	r.entradas[i].Tipo = e.Tipo
	r.entradas[i].URL = e.URL
	r.entradas[i].Extensiones = append([]string(nil), e.Extensiones...)
	// El adaptador vivo NO se toca: recrearlo en caliente dejaría dos
	// versiones del mismo motor sirviendo turnos a la vez (DECISIONS). El
	// cambio se aplica al reiniciar, y mientras tanto se marca.
	if _, aplicado := r.adaptadores[e.ID]; aplicado {
		r.iniciarMapas()
		r.pendientesDeReinicio[e.ID] = true
	}
	return r.guardar()
}

// Desactivar deja el motor registrado y visible pero no usable. No borra ni
// sus modelos registrados: desactivar es reversible, no es eliminar
// (SPEC-MODELO-MOTOR §Desactivar).
func (r *Registro) Desactivar(id string) error {
	return r.cambiarActivo(id, false)
}

// Reactivar vuelve a poner el motor disponible con los modelos que tenía
// registrados, sin volver a agregarlo (SPEC-MODELO-MOTOR §Reactivar).
func (r *Registro) Reactivar(id string) error {
	return r.cambiarActivo(id, true)
}

func (r *Registro) cambiarActivo(id string, activo bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indice(id)
	if i < 0 {
		return fmt.Errorf("llm: no hay ningún motor registrado con id %q", id)
	}
	r.entradas[i].Activo = activo
	// Desactivar y reactivar se aplican al momento: se destruye o se
	// reconstruye el adaptador vivo. Reactivar usa la configuración actual,
	// así que ya no queda nada pendiente de reinicio.
	r.iniciarMapas()
	delete(r.adaptadores, id)
	delete(r.pendientesDeReinicio, id)
	if activo {
		m, err := AdaptadorDeTipo(r.entradas[i].Tipo, r.entradas[i].URL)
		if err != nil {
			return err
		}
		r.adaptadores[id] = m
	}
	return r.guardar()
}

// Eliminar borra el motor completo: su configuración y sus modelos
// registrados. No elimina ni invalida ninguna sesión que lo usara —no hay
// integridad referencial entre la sesión y el registro, a propósito—: la
// sesión avisa de que su motor falta y espera (SPEC-MODELO-MOTOR §Eliminar,
// CONSTRAINTS sobre sessions.motor_id).
func (r *Registro) Eliminar(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indice(id)
	if i < 0 {
		return fmt.Errorf("llm: no hay ningún motor registrado con id %q", id)
	}
	r.entradas = append(r.entradas[:i], r.entradas[i+1:]...)
	delete(r.modelos, id)
	// Eliminar borra también su caché viva y su adaptador: la configuración y
	// sus datos registrados desaparecen juntos (SPEC-MODELO-MOTOR §Eliminar).
	delete(r.caches, id)
	delete(r.adaptadores, id)
	delete(r.pendientesDeReinicio, id)
	return r.guardar()
}

// Adaptador devuelve el adaptador VIVO del motor, construido la primera vez
// por su tipo cerrado y cacheado por `id`. Un motor desactivado, eliminado o
// con tipo desconocido no tiene adaptador: el llamante avisa con
// E_MOTOR_NO_DISPONIBLE y la sesión espera (SPEC-MODELO-MOTOR).
func (r *Registro) Adaptador(id string) (Motor, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indice(id)
	if i < 0 || !r.entradas[i].Activo {
		return nil, false
	}
	if m, ok := r.adaptadores[id]; ok {
		return m, true
	}
	if err := r.construirAdaptador(r.entradas[i]); err != nil {
		return nil, false
	}
	return r.adaptadores[id], true
}

// PendienteDeReinicio dice si el motor se editó después de estar aplicado, así
// que su cambio espera al próximo arranque (DECISIONS).
func (r *Registro) PendienteDeReinicio(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pendientesDeReinicio[id]
}

// construirAdaptador instancia el adaptador de una entrada por su tipo
// cerrado. Se llama con el candado tomado.
func (r *Registro) construirAdaptador(e Entrada) error {
	m, err := AdaptadorDeTipo(e.Tipo, e.URL)
	if err != nil {
		return err
	}
	r.iniciarMapas()
	r.adaptadores[e.ID] = m
	return nil
}

// PorID devuelve la entrada registrada con ese id.
func (r *Registro) PorID(id string) (Entrada, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indice(id)
	if i < 0 {
		return Entrada{}, false
	}
	return r.entradas[i], true
}

// Activos devuelve, en orden de registro, los motores activos.
func (r *Registro) Activos() []Entrada {
	r.mu.Lock()
	defer r.mu.Unlock()

	salida := make([]Entrada, 0, len(r.entradas))
	for _, e := range r.entradas {
		if e.Activo {
			salida = append(salida, e)
		}
	}
	return salida
}

// Todas devuelve, en orden de registro, todas las entradas incluidas las
// desactivadas: un motor desactivado permanece visible en la lista, que es lo
// que lo distingue de uno eliminado (SPEC-MODELO-MOTOR §Desactivar).
func (r *Registro) Todas() []Entrada {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Entrada(nil), r.entradas...)
}

// Modelos devuelve la caché `modelos_por_motor` del motor: el último catálogo
// que se vio. Puede estar desactualizada; al conectar manda lo que el motor
// responda (CONFIGURATION.md §8).
func (r *Registro) Modelos(id string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.modelos[id]...)
}

// RefrescarModelos guarda el catálogo que el motor acaba de declarar como su
// último conocido y persiste. El motor tiene que estar registrado: un motor
// eliminado no tiene caché que conservar.
func (r *Registro) RefrescarModelos(id string, modelos []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.indice(id) < 0 {
		return fmt.Errorf("llm: no hay ningún motor registrado con id %q", id)
	}
	if r.modelos == nil {
		r.modelos = map[string][]string{}
	}
	if len(modelos) == 0 {
		delete(r.modelos, id)
	} else {
		r.modelos[id] = append([]string(nil), modelos...)
	}
	return r.guardar()
}

// iniciarMapas crea los mapas internos si el registro se construyó a mano (los
// tests) en vez de con Cargar. Se llama con el candado tomado.
func (r *Registro) iniciarMapas() {
	if r.adaptadores == nil {
		r.adaptadores = map[string]Motor{}
	}
	if r.pendientesDeReinicio == nil {
		r.pendientesDeReinicio = map[string]bool{}
	}
}

// cacheDevuelve la caché viva del motor, creándola a la primera petición.
// Se llama con el candado tomado.
func (r *Registro) cacheDevuelve(id string) *cacheMotor {
	if r.caches == nil {
		r.caches = map[string]*cacheMotor{}
	}
	c, ok := r.caches[id]
	if !ok {
		c = nuevoCacheMotor()
		r.caches[id] = c
	}
	return c
}

// VentanaDelServidor devuelve la ventana cacheada que fija el servidor para
// este motor y si ya se leyó (false = aún no se preguntó; preguntar es cosa
// del llamante, que habla con el motor).
func (r *Registro) VentanaDelServidor(idMotor string) (ventana int, leida bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.caches[idMotor]
	if !ok {
		return 0, false
	}
	return c.ventanaServidor, c.ventanaLeida
}

// CachearVentanaDelServidor guarda la ventana que fija el servidor para este
// motor. Se cachea también cuando es 0 (el motor la declara por petición y no
// la fija él): leerla una sola vez es el punto de la caché.
func (r *Registro) CachearVentanaDelServidor(idMotor string, ventana int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c := r.cacheDevuelve(idMotor)
	c.ventanaServidor, c.ventanaLeida = ventana, true
}

// LargoDeContexto devuelve la ventana cacheada que declara el modelo EN ESTE
// MOTOR y si ya se cacheó (un 0 cacheado = el modelo no la declara).
func (r *Registro) LargoDeContexto(idMotor, modelo string) (largo int, cacheado bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.caches[idMotor]
	if !ok {
		return 0, false
	}
	largo, cacheado = c.contextos[modelo]
	return largo, cacheado
}

// CachearLargoDeContexto guarda la ventana que declara el modelo en este
// motor. Un 0 se guarda igual: «no la declara» también es un dato que no
// hace falta volver a preguntar.
func (r *Registro) CachearLargoDeContexto(idMotor, modelo string, largo int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheDevuelve(idMotor).contextos[modelo] = largo
}

// CachearModelos rellena de una pasada las ventanas que declaran los modelos
// que el motor acaba de listar: el listado trae la de todos, no solo la
// pedida.
func (r *Registro) CachearModelos(idMotor string, modelos []Modelo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c := r.cacheDevuelve(idMotor)
	for _, m := range modelos {
		c.contextos[m.Nombre] = m.ContextLength
	}
}

// Ficha devuelve la caché de capacidades del modelo en este motor y si ya se
// conoce. La ficha es del motor y el modelo: dos motores del mismo tipo no la
// comparten aunque sirvan el mismo nombre de modelo.
func (r *Registro) Ficha(idMotor, modelo string) (capacidades []string, conocida bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.caches[idMotor]
	if !ok {
		return nil, false
	}
	caps, ok := c.capacidades[modelo]
	return append([]string(nil), caps...), ok
}

// CachearFicha guarda la ficha de capacidades del modelo en este motor. Una
// ficha vacía NO se cachea: un fallo de red no debe fijar «no puede».
func (r *Registro) CachearFicha(idMotor, modelo string, capacidades []string) {
	if len(capacidades) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheDevuelve(idMotor).capacidades[modelo] = append([]string(nil), capacidades...)
}

// indice es la posición de `id` en las entradas, o -1. Se llama con el
// candado tomado.
func (r *Registro) indice(id string) int {
	for i, e := range r.entradas {
		if e.ID == id {
			return i
		}
	}
	return -1
}

// nombrarEntrada nombra una entrada para un aviso por su nombre visible y, si
// lo tiene, por su id; una entrada sin nombre no se señala como vacía.
func nombrarEntrada(e Entrada) string {
	switch {
	case e.Nombre != "" && e.ID != "":
		return e.Nombre + " (" + e.ID + ")"
	case e.Nombre != "":
		return e.Nombre
	case e.ID != "":
		return e.ID
	default:
		return "(sin nombre)"
	}
}

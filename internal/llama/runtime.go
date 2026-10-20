package llama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"localcli/internal/models"
)

// esperaMaxima es cuánto se aguarda a que el servidor responda tras arrancarlo.
const esperaMaxima = 60 * time.Second

// topeRegistro acota el archivo de salida del motor; al superarlo se recicla
// para que el diagnóstico no crezca sin fin.
const topeRegistro = 4 << 20

// Modelo es el modelo que se le pasa a `llama-server`: su nombre (la carpeta), la
// ruta de su GGUF y, si es multimodal, la de su proyector.
type Modelo struct {
	Nombre string
	Ruta   string
	// MMProj es la ruta del proyector multimodal (`--mmproj`). Vacío en un modelo
	// de solo texto.
	MMProj string
	// Contexto es la ventana de contexto con la que se arranca el servidor (`-c`):
	// el contexto real del modelo. 0 cae al valor por defecto del servidor.
	Contexto int
}

// Servidor gestiona el ciclo de vida de un único `llama-server`. Es seguro para
// uso concurrente.
type Servidor struct {
	binario  string
	baseURL  string
	host     string
	puerto   int
	contexto int

	mu      sync.Mutex
	modelo  Modelo
	proc    *ejecucion
	espera  time.Duration
	diag    *bufferLimitado
	rutaLog string
}

// ejecucion es un proceso arrancado y la señal de que terminó; `Wait` corre en
// segundo plano para reapear y no dejar zombis.
type ejecucion struct {
	cmd     *exec.Cmd
	hecho   chan struct{}
	archivo *os.File
	err     error
}

// lanzar arranca el proceso y espera su fin en segundo plano. Si hay archivo de
// registro, se cierra cuando el proceso termina.
func lanzar(cmd *exec.Cmd, archivo *os.File) (*ejecucion, error) {
	if err := cmd.Start(); err != nil {
		if archivo != nil {
			_ = archivo.Close()
		}
		return nil, err
	}
	e := &ejecucion{cmd: cmd, hecho: make(chan struct{}), archivo: archivo}
	go func() {
		e.err = cmd.Wait()
		if e.archivo != nil {
			_ = e.archivo.Close()
		}
		close(e.hecho)
	}()
	return e, nil
}

// vivo indica si el proceso sigue en marcha.
func (e *ejecucion) vivo() bool {
	select {
	case <-e.hecho:
		return false
	default:
		return true
	}
}

// esperar aguarda a que el proceso termine dentro del tope.
func (e *ejecucion) esperar(tope time.Duration) bool {
	select {
	case <-e.hecho:
		return true
	case <-time.After(tope):
		return false
	}
}

// motivo describe cómo terminó el proceso: código de salida o señal. Un
// SIGSEGV del motor se ve como «señal SIGSEGV», no como un simple «terminó».
func (e *ejecucion) motivo() string {
	if e.err == nil {
		return "código 0"
	}
	var salida *exec.ExitError
	if errors.As(e.err, &salida) {
		if estado, ok := salida.Sys().(syscall.WaitStatus); ok && estado.Signaled() {
			return "señal " + nombreDeSenal(estado.Signal())
		}
		if codigo := salida.ExitCode(); codigo >= 0 {
			return fmt.Sprintf("código %d", codigo)
		}
	}
	return e.err.Error()
}

// nombreDeSenal nombra las señales que de verdad se ven al caer un motor.
func nombreDeSenal(senal syscall.Signal) string {
	switch senal {
	case syscall.SIGSEGV:
		return "SIGSEGV"
	case syscall.SIGABRT:
		return "SIGABRT"
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGILL:
		return "SIGILL"
	case syscall.SIGBUS:
		return "SIGBUS"
	case syscall.SIGFPE:
		return "SIGFPE"
	default:
		return senal.String()
	}
}

// Nuevo construye el gestor. `binario` puede ir vacío para buscarlo en PATH;
// `baseURL` es la dirección `api`; `contexto` es el `-c` de arranque.
func Nuevo(binario, baseURL string, contexto int) (*Servidor, error) {
	host, puerto, err := HostPuerto(baseURL)
	if err != nil {
		return nil, err
	}
	if contexto <= 0 {
		contexto = 4096
	}
	return &Servidor{
		binario:  strings.TrimSpace(binario),
		baseURL:  strings.TrimRight(baseURL, "/"),
		host:     host,
		puerto:   puerto,
		contexto: contexto,
		espera:   esperaMaxima,
		diag:     nuevoBufferLimitado(8 * 1024),
	}, nil
}

// ConRegistro fija el archivo donde se guarda la salida completa del motor. Si
// no se puede abrir, el arranque sigue con la cola en memoria. Devuelve el
// propio servidor para encadenar la configuración.
func (s *Servidor) ConRegistro(ruta string) *Servidor {
	s.mu.Lock()
	s.rutaLog = strings.TrimSpace(ruta)
	s.mu.Unlock()
	return s
}

// Asegurar deja el servidor sirviendo `m` sin levantar un segundo proceso. Solo
// devuelve error cuando el arranque falla de forma insalvable; los motivos que
// no bloquean el harness (falta el binario, falta el modelo, no cabe en la
// máquina, puerto ocupado) salen como avisos accionables.
func (s *Servidor) Asegurar(ctx context.Context, m Modelo) ([]string, error) {
	var avisos []string

	binario, err := LocalizarBinario(s.binario)
	if err != nil {
		return append(avisos, err.Error()), nil
	}
	s.binario = binario

	if err := m.validar(); err != nil {
		return append(avisos, err.Error()), nil
	}
	if _, err := os.Stat(m.Ruta); err != nil {
		return append(avisos, fmt.Sprintf("llama: el modelo %q no está en disco: %s", m.Nombre, m.Ruta)), nil
	}
	if motivo := models.MotivoCabezaDraft(m.Ruta); motivo != "" {
		return append(avisos, fmt.Sprintf("llama: el modelo %q %s; no se arranca", m.Nombre, motivo)), nil
	}
	if strings.TrimSpace(m.MMProj) != "" {
		if _, err := os.Stat(m.MMProj); err != nil {
			avisos = append(avisos, fmt.Sprintf("llama: el proyector multimodal de %q no está en disco: %s", m.Nombre, m.MMProj))
		}
	}
	if tamano := tamanoDe(m.Ruta); tamano > 0 {
		if cabe, conocido := CabeEnMemoria(tamano); conocido && !cabe {
			avisos = append(avisos, fmt.Sprintf("llama: el modelo %q (%s) no cabe en la memoria de la máquina", m.Nombre, humano(tamano)))
		}
	}

	// Nuestro propio proceso ya sirve este modelo: nada que hacer.
	s.mu.Lock()
	corriendo := s.proc != nil && s.proc.vivo()
	actual := s.modelo
	s.mu.Unlock()
	if corriendo && actual.Nombre == m.Nombre && s.Disponible(ctx) {
		return avisos, nil
	}
	if corriendo {
		_ = s.Detener()
	}

	// Ya hay un servidor compatible levantado en el puerto: se reutiliza.
	if s.Disponible(ctx) && s.sirveModelo(ctx, m.Nombre) {
		s.fijarModelo(m)
		return avisos, nil
	}

	if ocupado, err := puertoOcupado(s.host, s.puerto); err != nil {
		return avisos, err
	} else if ocupado {
		return append(avisos, fmt.Sprintf("llama: el puerto %d está ocupado por otro proceso; no se arranca llama-server", s.puerto)), nil
	}

	if err := s.Arrancar(ctx, m); err != nil {
		return avisos, err
	}
	if err := s.EsperarDisponible(ctx); err != nil {
		return avisos, err
	}
	return avisos, nil
}

// Arrancar levanta el proceso con el modelo dado y espera a que responda.
func (s *Servidor) Arrancar(ctx context.Context, m Modelo) error {
	if err := m.validar(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proc != nil && s.proc.vivo() {
		return fmt.Errorf("llama: ya hay un llama-server arrancado")
	}
	s.proc = nil

	s.diag.Reiniciar()
	salida, archivo := s.abrirRegistro(m)
	cmd := exec.Command(s.binario, s.argsDeArranque(m)...)
	cmd.Stdout = salida
	cmd.Stderr = salida
	e, err := lanzar(cmd, archivo)
	if err != nil {
		return fmt.Errorf("llama: no se pudo arrancar llama-server: %w", err)
	}
	s.proc = e
	s.modelo = m
	return nil
}

// Detener apaga el servidor de forma limpia: SIGTERM y, si no cede, SIGKILL. No
// deja procesos huérfanos.
func (s *Servidor) Detener() error {
	s.mu.Lock()
	e := s.proc
	s.proc = nil
	s.mu.Unlock()
	if e == nil {
		return nil
	}
	if e.vivo() {
		_ = e.cmd.Process.Signal(syscall.SIGTERM)
		if !e.esperar(5 * time.Second) {
			_ = e.cmd.Process.Kill()
			e.esperar(2 * time.Second)
		}
	}
	return nil
}

// Reiniciar detiene el servidor y lo vuelve a arrancar con el modelo nuevo. Si
// el modelo nuevo no es arrancable, avisa sin tocar el que ya estaba sirviendo.
func (s *Servidor) Reiniciar(ctx context.Context, m Modelo) ([]string, error) {
	if motivo := models.MotivoCabezaDraft(m.Ruta); motivo != "" {
		return []string{fmt.Sprintf("llama: el modelo %q %s; no se arranca", m.Nombre, motivo)}, nil
	}
	_ = s.Detener()
	return s.Asegurar(ctx, m)
}

// Disponible indica si el servidor responde en la dirección de la API.
func (s *Servidor) Disponible(ctx context.Context) bool {
	pctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(pctx, http.MethodGet, s.baseURL+"/v1/models", nil)
	if err != nil {
		return false
	}
	resp, err := s.http().Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return resp.StatusCode == http.StatusOK
}

// EsperarDisponible aguarda a que el proceso responda. Si el proceso muere antes,
// devuelve el motivo con su salida como pista.
func (s *Servidor) EsperarDisponible(ctx context.Context) error {
	final := time.Now().Add(s.espera)
	for time.Now().Before(final) {
		if s.Disponible(ctx) {
			return nil
		}
		s.mu.Lock()
		e := s.proc
		s.mu.Unlock()
		if e != nil && !e.vivo() {
			s.mu.Lock()
			s.proc = nil
			s.mu.Unlock()
			return fmt.Errorf("llama: llama-server terminó antes de responder (%s)%s", e.motivo(), s.pista())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return fmt.Errorf("llama: llama-server no respondió en %s%s", s.espera, s.pista())
}

// Activo devuelve el modelo que sirve el servidor gestionado, si lo hay.
func (s *Servidor) Activo() (Modelo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proc == nil || !s.proc.vivo() {
		return Modelo{}, false
	}
	return s.modelo, true
}

// argsDeArranque compone los argumentos: el GGUF, su alias (la carpeta), el
// puerto de la API, la ventana de contexto y, si es multimodal, el proyector. La
// ventana es la del modelo (`m.Contexto`) cuando la trae; si no, la del servidor.
func (s *Servidor) argsDeArranque(m Modelo) []string {
	contexto := s.contexto
	if m.Contexto > 0 {
		contexto = m.Contexto
	}
	args := []string{
		"--model", m.Ruta,
		"--alias", m.Nombre,
		"--port", strconv.Itoa(s.puerto),
		"-c", strconv.Itoa(contexto),
	}
	if strings.TrimSpace(m.MMProj) != "" {
		args = append(args, "--mmproj", m.MMProj)
	}
	return args
}

// abrirRegistro prepara la salida del proceso: la cola en memoria y, si hay
// ruta de registro, el archivo completo. Un archivo que no se puede abrir no
// impide arrancar el motor.
func (s *Servidor) abrirRegistro(m Modelo) (io.Writer, *os.File) {
	if s.rutaLog == "" {
		return s.diag, nil
	}
	if err := os.MkdirAll(filepath.Dir(s.rutaLog), 0o755); err != nil {
		return s.diag, nil
	}
	f, err := os.OpenFile(s.rutaLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return s.diag, nil
	}
	if info, err := f.Stat(); err == nil && info.Size() > topeRegistro {
		_ = f.Truncate(0)
	}
	fmt.Fprintf(f, "\n=== %s · %s · %s ===\n", time.Now().Format(time.RFC3339), m.Nombre,
		strings.Join(s.argsDeArranque(m), " "))
	return io.MultiWriter(s.diag, f), f
}

// sirveModelo indica si el servidor ya levantado lista ese nombre de modelo.
func (s *Servidor) sirveModelo(ctx context.Context, nombre string) bool {
	ids, err := s.modelosServidos(ctx)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == nombre || models.NombreDe(id) == nombre {
			return true
		}
	}
	return false
}

// modelosServidos lee los ids que reporta `GET /v1/models`.
func (s *Servidor) modelosServidos(ctx context.Context) ([]string, error) {
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(pctx, http.MethodGet, s.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llama: el servidor respondió %s", resp.Status)
	}
	var bruto struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Nombre string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bruto); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(bruto.Data)+len(bruto.Models))
	for _, m := range bruto.Data {
		ids = append(ids, m.ID)
	}
	for _, m := range bruto.Models {
		ids = append(ids, m.Nombre)
	}
	return ids, nil
}

func (s *Servidor) fijarModelo(m Modelo) {
	s.mu.Lock()
	s.modelo = m
	s.mu.Unlock()
}

// pista devuelve la salida capturada del proceso para el mensaje de error.
func (s *Servidor) pista() string {
	texto := ""
	if s.diag != nil {
		texto = strings.TrimSpace(s.diag.String())
	}
	if s.rutaLog == "" {
		if texto == "" {
			return ""
		}
		return ": " + texto
	}
	if texto == "" {
		return "; salida completa en " + s.rutaLog
	}
	return "; salida completa en " + s.rutaLog + ": " + texto
}

// http es el cliente con el que se sondea el servidor.
func (s *Servidor) http() *http.Client {
	return &http.Client{Timeout: time.Second}
}

// validar exige nombre y ruta: sin ellos no hay nada que arrancar.
func (m Modelo) validar() error {
	if strings.TrimSpace(m.Ruta) == "" {
		return errors.New("llama: no hay modelo seleccionado")
	}
	if strings.TrimSpace(m.Nombre) == "" {
		return errors.New("llama: el modelo no tiene nombre (carpeta)")
	}
	return nil
}

// puertoOcupado comprueba si el puerto ya está tomado.
func puertoOcupado(host string, puerto int) (bool, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(puerto)))
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return true, nil
		}
		return false, fmt.Errorf("llama: no se pudo comprobar el puerto %d: %w", puerto, err)
	}
	_ = ln.Close()
	return false, nil
}

// tamanoDe devuelve el tamaño del archivo, o 0 si no se pudo leer.
func tamanoDe(ruta string) int64 {
	info, err := os.Stat(ruta)
	if err != nil {
		return 0
	}
	return info.Size()
}

// bufferLimitado guarda solo los últimos bytes escritos, para diagnosticar sin
// crecer sin fin.
type bufferLimitado struct {
	mu    sync.Mutex
	tope  int
	datos bytes.Buffer
}

func nuevoBufferLimitado(tope int) *bufferLimitado {
	return &bufferLimitado{tope: tope}
}

func (b *bufferLimitado) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, _ := b.datos.Write(p)
	if b.datos.Len() > b.tope {
		sobra := b.datos.Len() - b.tope
		b.datos.Next(sobra)
	}
	return n, nil
}

// Reiniciar vacía la cola antes de un arranque nuevo.
func (b *bufferLimitado) Reiniciar() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.datos.Reset()
}

func (b *bufferLimitado) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.datos.String()
}

// profile.go — T-B005-05: perfil de hardware y cabida de modelos.
//
// Fuente de verdad: SPEC-OLLAMA-PERFIL §Reglas de negocio:
//   - "El perfil por defecto se calcula para 4 GB de VRAM y 16 GB de RAM."
//   - "La herramienta muestra qué modelos caben en el hardware detectado
//     antes de que el usuario elija."
//   - "Si el modelo elegido no cabe, avisa y no lo carga en silencio" →
//     AvisoNoCabe con E_MODEL_TOO_BIG; NO es un fallo: la carga continúa
//     (ERRORS.md: "lo carga igual, sin fallar en silencio").
//   - "El tamaño de contexto se limita para que quepa junto con el modelo."
package ollama

import (
	"os"
	"regexp"
	"strconv"
)

// Hardware describe la máquina detectada. Bytes donde quepa decir bytes.
type Hardware struct {
	RAMTotalBytes  int64
	VRAMTotalBytes int64 // 0 si no hay GPU detectable
}

// PerfilPorDefecto según SPEC-OLLAMA-PERFIL: objetivo 4 GB VRAM / 16 GB RAM.
func PerfilPorDefecto() Hardware {
	return Hardware{
		RAMTotalBytes:  16 * GiB,
		VRAMTotalBytes: 4 * GiB,
	}
}

const (
	KiB int64 = 1024
	MiB int64 = 1024 * KiB
	GiB int64 = 1024 * MiB
)

// DetectarHardware lee la RAM total de /proc/meminfo (Linux). La VRAM no tiene
// una fuente estándar sin dependencias (nvidia-smi etc.), así que se queda en
// 0 = desconocida y el llamante puede mezclar con PerfilPorDefecto u
// configuración del usuario. Nunca falla: si no puede leer, devuelve ceros.
func DetectarHardware() Hardware {
	h := Hardware{}
	datos, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return h
	}
	h.RAMTotalBytes = parsearMeminfo(string(datos))
	return h
}

var reMemTotal = regexp.MustCompile(`MemTotal:\s+(\d+)\s+kB`)

func parsearMeminfo(contenido string) int64 {
	m := reMemTotal.FindStringSubmatch(contenido)
	if m == nil {
		return 0
	}
	kb, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0
	}
	return kb * KiB
}

// EstimacionMB aproxima cuánto ocupa un modelo: peso en disco + margen de
// contexto/activaciones. Regla simple documentada aquí a propósito: sin SDK
// ni heurísticas por familia en la primera versión; el aviso al usuario dice
// "estimados".
func EstimacionMB(tamanoBytes int64) int64 {
	pesoMB := tamanoBytes / MiB
	return pesoMB + pesoMB/8 + 512 // ~12.5% + 512 MB de colchón
}

// Cabe indica si el modelo entra en la VRAM declarada. Con VRAM desconocida
// (0) responde false pero el avisador lo trata como "no comprobable": ver
// AvisarSiNoCabe.
func (h Hardware) Cabe(m Modelo) bool {
	if h.VRAMTotalBytes <= 0 || m.TamanoBytes <= 0 {
		return false
	}
	return EstimacionMB(m.TamanoBytes)*MiB <= h.VRAMTotalBytes
}

// ClasificarModelos particiona la lista en los que caben y los que no, para
// mostrarlos ANTES de que el usuario elija (regla dura del spec). Modelos sin
// tamaño reportado van a "desconocidos" — informamos, no decidimos.
func (h Hardware) ClasificarModelos(lista []Modelo) (caben, noCaben, desconocidos []Modelo) {
	for _, m := range lista {
		switch {
		case m.TamanoBytes <= 0 || h.VRAMTotalBytes <= 0:
			desconocidos = append(desconocidos, m)
		case h.Cabe(m):
			caben = append(caben, m)
		default:
			noCaben = append(noCaben, m)
		}
	}
	return
}

// AvisoNoCabe devuelve el AVISA tipado (E_MODEL_TOO_BIG) si el modelo elegido
// no cabe en la VRAM del perfil. Devuelve nil si cabe o si no se puede
// comprobar (VRAM/tamaño desconocidos: mejor no gritar). El flujo de carga
// sigue adelante con el aviso: nunca aborta (ERRORS.md [20]).
func (h Hardware) AvisoNoCabe(m Modelo) error {
	if m.TamanoBytes <= 0 || h.VRAMTotalBytes <= 0 {
		return nil
	}
	if h.Cabe(m) {
		return nil
	}
	return nuevoModeloNoCabe(m.Nombre, EstimacionMB(m.TamanoBytes), h.VRAMTotalBytes/MiB)
}

// ContextoLimitadoTokens recorta el tamaño de contexto para que quepa junto
// con el modelo cargado (regla del spec + INTEGRATIONS §5 "el límite de
// contexto del modelo manda"). Aproximación: 1 token ≈ 4 bytes de pesos KV;
// presupuesto = RAM libre tras el modelo, mitad para contexto.
func (h Hardware) ContextoLimitadoTokens(m Modelo, maxDelModelo int) int {
	const bytesPorToken = 4
	reserva := m.TamanoBytes
	disponible := h.RAMTotalBytes - reserva
	if disponible < 0 {
		disponible = 0
	}
	presupuesto := int(disponible / 2 / bytesPorToken)
	if presupuesto > maxDelModelo {
		presupuesto = maxDelModelo
	}
	// Suelo práctico: menos de 512 tokens no deja trabajar al nodo de
	// contexto; mejor avisar arriba (recorte registrado) que callar.
	if presupuesto < 512 {
		presupuesto = 512
	}
	return presupuesto
}

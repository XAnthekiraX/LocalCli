// Package git — el estado del repositorio del proyecto para el panel.
//
// Fuente de verdad: ai/docs/specs/SPEC-INTERFAZ.md §Zonas 3 (el dato «Git»
// del sidebar: rama activa, y si el árbol tiene cambios sin confirmar) y sus
// reglas de §Pie del sidebar (sin git inicializado se pinta «sin iniciar»).
//
// Es una lectura de solo consulta, no la herramienta `ejecutar_comando`: aquí no
// hay aprobación, ni sandbox, ni límite de salida, porque el harness lee su
// propio repositorio para pintarlo y no ejecuta nada del proyecto. Llama al
// binario `git` en vez de arrastrar una biblioteca, como ya hace el resto del
// proyecto con las herramientas del terminal.
//
// No guarda nada: el estado se lee cuando se necesita y se entrega como dato. No
// va a la base, porque describe el repositorio y no la sesión.
package git

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// limite es el tiempo que se le da a cada llamada. `git status` sobre un árbol
// grande puede tardar, pero si se pasa es que el disco va mal y pintar «sin
// iniciar» es mejor que colgar el arranque.
const limite = 3 * time.Second

// Estado devuelve la rama activa del repositorio que contiene `dir` y si el
// árbol tiene cambios sin confirmar.
//
// La rama vacía es el caso de «no hay git aquí»: o el binario no está
// instalado, o `dir` no está dentro de un repositorio, o git no responde a
// tiempo. Quien llama la muestra como «sin iniciar». `limpio` solo significa algo
// cuando la rama no viene vacía.
func Estado(dir string) (rama string, limpio bool) {
	// `--is-inside-work-tree` responde «true» dentro de un repositorio y sale
	// con error fuera de él, que incluye el caso de que git no esté instalado.
	if salida, err := correr(dir, "rev-parse", "--is-inside-work-tree"); err != nil ||
		strings.TrimSpace(salida) != "true" {
		return "", false
	}
	// `status --porcelain --branch` da las dos cosas de una vez: la primera
	// línea es la cabecera `## rama...` y el resto son los cambios pendientes.
	salida, err := correr(dir, "status", "--porcelain", "--branch")
	if err != nil {
		return "", false
	}
	var cambios int
	visto := false
	for _, l := range strings.Split(salida, "\n") {
		l = strings.TrimRight(l, "\r")
		if l == "" {
			continue
		}
		if !visto {
			// La cabecera `##` siempre va la primera; si no aparece, la salida
			// no es la que se espera y no hay rama que enseñar.
			rama, visto = ramaDeCabecera(l), true
			continue
		}
		cambios++
	}
	if rama == "" {
		return "", false
	}
	return rama, cambios == 0
}

// ramaDeCabecera saca la rama de la línea `##` que pone `status --porcelain
// --branch`. Hay tres formas según el estado del repositorio:
//
//	## main...origin/main          rama normal, con su remoto si lo tiene
//	## No commits yet on main      repositorio recién inicializado, sin commits
//	## HEAD (no branch)            HEAD desligado: la referencia ya no apunta a
//	                              ninguna rama, así que el nombre que trae no
//	                              sirve y se devuelve vacía
func ramaDeCabecera(linea string) string {
	if !strings.HasPrefix(linea, "## ") {
		return ""
	}
	cabecera := strings.TrimPrefix(linea, "## ")
	if i := strings.Index(cabecera, "..."); i >= 0 {
		// La parte antes de los puntos es la rama local; lo de detrás es su
		// remoto y la diferencia con él.
		cabecera = cabecera[:i]
	}
	if _, sinCommits, ok := strings.Cut(cabecera, "No commits yet on "); ok {
		return sinCommits
	}
	if cabecera == "HEAD (no branch)" {
		return ""
	}
	return strings.TrimSpace(cabecera)
}

// correr ejecuta un `git` de solo lectura en `dir` y devuelve su salida. No
// propaga los errores: quien llama solo necesita saber si hubo respuesta, y un
// fallo de git se traduce en «sin iniciar», no en un error de arranque.
func correr(dir string, args ...string) (string, error) {
	// `-C dir` evita tener que cambiar de directorio: no se toca el directorio
	// de trabajo del proceso por una lectura.
	ctx, cancelar := context.WithTimeout(context.Background(), limite)
	defer cancelar()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	salida, err := cmd.Output()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return string(salida), err
}

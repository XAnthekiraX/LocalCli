package exec

import (
	"context"
	"strings"
	"testing"

	"localcli/internal/tools"
)

// Una herramienta del usuario inocua corre sin preguntar; solo la destructiva
// o la que toca rutas fuera de la carpeta pide confirmación.
func TestHerramientaDeUsuarioSinShellYConAprobacion(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}

	saluda := tools.HerramientaUsuario{Nombre: "saluda", Descripcion: "saluda", Equipo: []string{"echo", "a;b"}}
	salida, err := e.HandlerUsuario(saluda)(context.Background(), map[string]any{"x": 1}, tools.Contexto{})
	if err != nil {
		t.Fatalf("no debería fallar: %v", err)
	}
	if !strings.Contains(salida.Salida, "a;b") {
		t.Fatalf("un argumento con `;` es un argumento literal: %q", salida.Salida)
	}

	borra := tools.HerramientaUsuario{Nombre: "borra", Descripcion: "borra", Equipo: []string{"rm", "f.txt"}}
	declinador := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	res, err := e.HandlerUsuario(borra)(context.Background(), map[string]any{}, declinador)
	if err != nil {
		t.Fatalf("declinar no es error de Go: %v", err)
	}
	if !strings.Contains(res.Error, tools.CodApprovalDeclined) {
		t.Fatalf("una herramienta destructiva declinada debería dar E_APPROVAL_DECLINED: %q", res.Error)
	}
}

// Sin mecanismo de aprobación, una herramienta destructiva no corre.
func TestHerramientaDeUsuarioPeligrosaSinAprobadorNoCorre(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	borra := tools.HerramientaUsuario{Nombre: "borra", Descripcion: "borra", Equipo: []string{"rm", "f.txt"}}
	res, err := e.HandlerUsuario(borra)(context.Background(), map[string]any{}, tools.Contexto{})
	if err != nil {
		t.Fatalf("sin mecanismo no es error de Go: %v", err)
	}
	if !strings.Contains(res.Error, tools.CodNeedsApproval) {
		t.Fatalf("debería dar E_NEEDS_APPROVAL: %q", res.Error)
	}
}

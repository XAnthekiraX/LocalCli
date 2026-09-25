package main

import "testing"

// TestNewRootModel — humo del stack: el modelo raíz se construye y cumple tea.Model.
func TestNewRootModel(t *testing.T) {
	m := newRootModel("LocalCli")
	if m.Init() != nil {
		t.Fatal("Init debe devolver nil en el modelo provisional")
	}
	if got := m.(rootModel).View(); got == "" {
		t.Fatal("View no debe devolver vacío")
	}
}

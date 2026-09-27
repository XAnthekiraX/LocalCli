// Package agent — módulo agent de LocalCli.
//
// Carga las definiciones de agente desde JSON: prompt, permisos y catálogo de
// herramientas (derivado). El agente base es un JSON editable, no código. Corre
// el ciclo conversacional (LLM → herramienta → resultado → LLM) y despacha las
// herramientas a tools; no las ejecuta por su cuenta.
package agent

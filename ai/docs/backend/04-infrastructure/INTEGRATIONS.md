# INTEGRATIONS — Servicios externos

Dos integraciones: Ollama, en local, y la búsqueda por internet. Ninguna otra sale de la máquina. Ver [[backend/03-security/SECURITY]] para las fronteras de seguridad.

## 1. APIs y servicios externos

### Ollama

- **Propósito:** es el modelo. Genera las respuestas, el razonamiento y las decisiones de contexto. Sin Ollama no hay respuestas; el harness es un cliente de él.
- **Dónde:** API HTTP en local, por defecto `http://localhost:11434`.
- **Streaming:** obligatorio. Peticiones en streaming para poder mostrar el razonamiento mientras llega, que es el requisito de la interfaz. Ver [[specs/SPEC-INTERFAZ]].
- **Perfil de hardware:** el harness detecta la máquina, avisa si el modelo elegido no cabe en la VRAM y limita el contexto. El modelo lo elige el usuario, no el harness. Ver [[specs/SPEC-OLLAMA-PERFIL]].
- **Concurrencia:** como las sesiones comparten un único modelo cargado, sus respuestas se serializan: mientras una genera, la otra espera. Es una consecuencia del hardware, no un defecto del diseño. Ver [[backend/DECISIONS]] para el mecanismo de serialización, aún abierto.

### Búsqueda en internet

- **Propósito:** consultar documentación de librerías mientras se planifica, y verificar versiones y APIs. Es la única integración que hace salir información de la máquina.
- **Qué sale:** solo la consulta que redacta el modelo. Nunca contenido del proyecto, documentación, historial ni tareas. Si la búsqueda necesita el proyecto, el agente te lo pide a ti.
- **Qué vuelve:** `buscar_en_internet` devuelve resultados (título, dirección, fragmento); `abrir_pagina` devuelve el contenido de una página. Ese contenido entra al presupuesto de contexto, se recorta y se audita igual que el resto.
- **Cómo se habilita:** no viene activada; el usuario la habilita. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 2. Webhooks

No aplican. No hay servidor, ni entradas HTTP, ni notificaciones entrantes. LocalCli no expone nada que un tercero pueda llamar.

## 3. SDKs y librerías

| Dependencia | Para qué | Nota |
|---|---|---|
| Cliente HTTP de la biblioteca estándar | Hablar con Ollama e internet | Sin SDK de terceros |
| Landlock (syscall del kernel) | Bloqueo estructural de escritura en la terminal | Linux 5.13+; en otros sistemas no está |
| `modernc.org/sqlite` | Acceso a SQLite sin cgo | Solo lo usa `store` |
| Bubble Tea + Lip Gloss | La TUI | Los usa `tui`, no el motor |

## 4. Credenciales y configuración requerida

Ninguna. Ollama corre en local sin credenciales, y la búsqueda por internet no pide clave de API. LocalCli no guarda ni pide secretos. Si algún día una integración los pidiera, se documentaría aquí.

Lo que sí se necesita en la máquina, pero no es una credencial: Ollama instalado y corriendo, y Landlock disponible si se quiere el aislamiento fuerte. Ver [[backend/04-infrastructure/CONFIGURATION]].

## 5. Contratos externos

### Contrato con Ollama

- El harness espera respuestas en streaming, con el razonamiento distinguible del texto final. De ahí sale el razonamiento en vivo de la interfaz.
- El harness detecta el hardware y valida el modelo antes de cargarlo. Si un modelo no cabe en la VRAM, avisa y lo carga en RAM, más lento, sin fallar en silencio.
- El límite de contexto del modelo manda: el nodo de contexto recorta para que lo entregado quepa. Ver [[backend/01-domain/DOMAIN]].

Si el contrato de streaming de Ollama cambiara, el módulo afectado es `ollama`, y el resto no debería enterarse.

### Contrato con la búsqueda

- La búsqueda devuelve solo lo público de internet.
- El contenido que vuelve es texto sin trust: entra al contexto como cualquier otro documento, sujeto al mismo recorte y auditoría. Ningún agente lo vuelca sin procesarlo.

## Referencias

- [[specs/SPEC-OLLAMA-PERFIL]] — modelo, hardware y límites.
- [[specs/SPEC-TOOLS]] — las herramientas de internet y sus límites.
- [[backend/03-security/SECURITY]] — qué sale y qué no.
- [[backend/04-infrastructure/CONFIGURATION]] — cómo se habilita internet.
- [[backend/04-infrastructure/EVENTS]] — cómo se notifica el streaming.

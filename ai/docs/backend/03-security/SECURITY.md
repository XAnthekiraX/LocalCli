# SECURITY — Seguridad y permisos

`LocalCli` es local y personal: no hay login, ni usuarios, ni sesiones de autenticación, ni endpoints que proteger de outsiders. La seguridad aquí es de otro tipo: se trata de que **nada de lo que hace el modelo toque tu proyecto ni tu máquina sin que tú lo hayas aprobado**, y de que un proceso hijo no pueda saltarse esa regla.

## 1. Autenticación

No hay autenticación, y no es una omisión. El proceso se abre en tu terminal, en tu máquina, con tus permisos. No hay red, así que no hay a quién autenticarse.

Lo que sí hay es **identidad de agente**: cada petición al motor viene de un agente (`plan` o `build`) y ese agente tiene un conjunto fijo de herramientas. La "autenticación" del modelo es saber qué agente es, no quién es el usuario. Ver [[backend/02-interfaces/TOOLS]].

## 2. Autorización

No hay roles de usuario. La autorización se resuelve en tres capas, y por eso es auditable: puedes leer en un solo sitio quién pidió, quién autorizó y quién ejecutó.

| Capa | Módulo | Qué decide |
|---|---|---|
| Qué puede pedir el agente | `agent` | Qué herramientas tiene `plan` (solo lectura) y `build` (todo) |
| Si la petición está permitida | `tools` | Que la herramienta exista y que el agente la tenga |
| Si el efecto se aplica | `fileops`, `exec` | La aprobación concreta, la frontera de rutas, la lista blanca y Landlock |

La garantía central: **`plan` no tiene herramientas de escritura**. No es que las tenga bloqueadas, es que no existen para él. Por eso la garantía de que nada se escribe sin propuesta y aprobación previa no depende de que alguien recuerde comprobar un permiso. Ver [[backend/DECISIONS]].

**Una aprobación vale para el cambio propuesto, no para lo que siga.** Si `build` necesita algo que `plan` no propuso, vuelve a preguntar.

## 3. Protección de las operaciones

No hay endpoints. Lo que hay son operaciones, y cada una tiene su regla.

### Toda escritura pasa por aprobación

Sin excepción: ni dentro de la carpeta del proyecto, ni fuera, ni dentro de un flujo en curso. Ver [[specs/SPEC-ARCHIVOS]].

### Frontera de rutas

- Toda ruta es relativa a la carpeta abierta del proyecto.
- Dentro de la carpeta: accesible, pero escribir sigue pidiendo aprobación.
- Fuera de la carpeta: hace falta permiso **y** la explicación del agente sobre por qué busca eso. La explicación queda visible para ti.

### Terminal bloqueada estructuralmente

La terminal tiene dos controles independientes:

1. **Lista blanca:** solo compilar, probar, revisar estilo y tipos, y ver estado/diferencias/historial de git corren sin preguntar. Cualquier otro comando pide aprobación.
2. **Bloqueo de escritura:** la terminal no puede crear, editar ni borrar archivos del proyecto, y la garantía es del kernel (Landlock), no del texto del comando. Por eso `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o `sh -c` con redirección no la esquivan.

La terminal tampoco puede descartar cambios del repositorio, hacer commits ni subir cambios.

## 4. Frontera de internet

Las dos herramientas de internet son las únicas que hacen salir información de la máquina, y ahí la regla es estrecha:

- **Solo sale la consulta que redacta el modelo.** Nunca el contenido de tus archivos, de la documentación, del historial ni de tus tareas.
- **Si una búsqueda necesita el proyecto para ser útil, el agente te lo pide a ti** en lugar de mandarlo.
- **Lo que vuelve entra al mismo presupuesto de contexto** que el resto y queda registrado en la auditoría, sujeto al mismo recorte. Ningún agente vierte el contenido de una página sin procesarlo. Ver [[specs/SPEC-TOOLS]].

## 5. Sanitización y datos sensibles

- El modelo solo recibe lo que el nodo de contexto decide. El contenido de los archivos que se entregan se lee antes, y lo que sale se registra. Ver [[backend/01-domain/DOMAIN]].
- No se envían datos personales del proyecto a internet por la vía de las herramientas de búsqueda.
- `change_history` guarda el antes y el después de cada cambio aplicado, lo que incluye contenido de tus archivos. Es un registro local en tu máquina, y existe para que puedas revertir y auditar, no para compartir.

## 6. Riesgos relevantes

- **Rate limiting:** no aplica. No hay red pública. Lo equivalente es el presupuesto de contexto, que recorta la entrada y evita que una respuesta o un comando desborden. Ver [[specs/SPEC-OLLAMA-PERFIL]].
- **CORS:** no aplica. No hay navegador ni servidor.
- **Secrets:** Ollama corre en local, sin credenciales. La búsqueda por internet es la única salida y solo lleva la consulta. LocalCli no guarda ni pide claves de API.
- **Aislamiento en sistemas sin Landlock:** en sistemas que no son Linux, o en Linux sin Landlock, la terminal no tiene el bloqueo estructural. La garantía es más débil y queda documentada como tal; el proyecto lo asume. Ver [[backend/04-infrastructure/CONFIGURATION]].
- **El modelo no es de fiar para decidir permisos.** `plan` y `build` tienen herramientas fijas, y el permiso lo comprueban módulos, no el modelo. El modelo no puede concederse permisos.
- **Confiar en el texto del comando es un error.** Cualquier intento de validar la terminal leyendo el comando es frágil por diseño; por eso el bloqueo es estructural.

## Referencias

- [[specs/SPEC-ARCHIVOS]] — reglas de permiso sobre archivos.
- [[specs/SPEC-TOOLS]] — el catálogo y los controles de la terminal.
- [[backend/02-interfaces/TOOLS]] — el reparto `plan`/`build`.
- [[backend/DECISIONS]] — por qué Landlock y por qué `plan` no escribe.
- [[backend/04-infrastructure/INTEGRATIONS]] — la integración de Ollama e internet.

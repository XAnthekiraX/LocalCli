# CONFIGURATION — Configuración del backend

`LocalCli` no tiene configuración obligatoria. Arranca y funciona. Casi todo se deriva solo: dónde está el proyecto, qué modelos hay disponibles y qué modelos caben en tu máquina.

## 1. El comando

`localcli` es el comando para arrancar la aplicación. Se instala una vez y queda disponible en todo el ordenador, así que lo ejecutas desde cualquier carpeta:

```
localcli
```

La carpeta desde la que lo ejecutas **es** el proyecto. No hay que pasar la ruta como argumento ni elegirla en un menú.

## 2. Variables de entorno

Ninguna es obligatoria. No hay variables que apliques por omisión.

| Variable | Propósito | Por defecto | Obligatoria |
|---|---|---|---|
| `LOCALCLI_DB_PATH` | Sobrescribe la ruta del archivo SQLite del proyecto | Derivada de la carpeta de ejecución | No |
| `LOCALCLI_CONTEXT_LIMIT` | Limita los tokens de contexto que se respetan al recortar | El del modelo elegido | No |
| `LOCALCLI_ALLOW_INTERNET` | Permite las herramientas de internet | Desactivado | No |

`LOCALCLI_DB_PATH` existe solo para casos raros, como trabajar con la base en otro sitio. En el uso normal no hace falta.

Sobre `LOCALCLI_ALLOW_INTERNET`: es la única integración que saca información de la máquina, así que no viene activada. El usuario la habilita. Ver [[backend/03-security/SECURITY]].

## 3. El modelo: se detecta, no se configura

**No hay variable de entorno para el modelo, y no hay modelo por defecto.** El flujo es:

1. `LocalCli` ya está conectado a Ollama. No hay que configurarlo a mano.
2. Extrae los modelos disponibles con `ollama list`.
3. La interfaz muestra esa lista para que el usuario elija.
4. El harness comprueba si el modelo elegido cabe en la máquina y avisa si no.

El modelo lo elige el usuario, siempre. Ver [[specs/SPEC-OLLAMA-PERFIL]] y [[backend/04-infrastructure/INTEGRATIONS]].

## 4. Rutas

Todo se deriva de la carpeta desde la que se ejecuta `localcli`:

| Qué | Dónde |
|---|---|
| Carpeta del proyecto | La carpeta desde la que se ejecuta `localcli` |
| Archivo SQLite | Derivado de esa carpeta |
| Documentación | `ai/docs/` dentro del proyecto |
| TODO de trabajo | `ai/tasks/` dentro del proyecto |
| Archivos y carpetas | Todo lo que cuelgue de la carpeta del proyecto |

El proyecto es la carpeta abierta. El chat de una carpeta nunca aparece en otra. Ver [[specs/SPEC-SESIONES]].

La ruta exacta del archivo SQLite es una decisión aún abierta en [[backend/DECISIONS]]; `LOCALCLI_DB_PATH` permite ajustarla sin esperar a cerrarla.

## 5. Configuración por ambiente

No hay ambientes de servidor. Hay tres formas de ejecutar:

| Modo | Para qué | Base de datos | Internet |
|---|---|---|---|
| Normal | Uso diario | Un archivo por proyecto | Solo si se habilita |
| Pruebas | Tests automatizados | Base temporal, aislada, que se destruye al terminar | Desactivado |
| Sin aislamiento fuerte | Sistema sin Landlock | Igual que el normal | Igual que el normal |

El tercer modo no es un flag: depende de si Landlock está disponible en el sistema. Si no lo está, el harness lo avisa y la garantía de la terminal es más débil, sin bloquear el uso. Ver [[backend/04-infrastructure/INTEGRATIONS]].

## 6. Servicios requeridos

| Servicio | Necesario para | Si no está |
|---|---|---|
| Ollama, corriendo en local | Generar respuestas y listar los modelos disponibles | El harness no puede generar nada; la interfaz sigue viva |
| Landlock (Linux 5.13+) | El bloqueo estructural de escritura en la terminal | La terminal sigue funcionando, con garantía más débil, y el harness lo dice |

No hay más servicios. No hay base de datos que levantar, ni migraciones que aplicar a mano, ni cuentas que crear. Ver [[specs/SPEC-OLLAMA-PERFIL]].

## Referencias

- [[backend/04-infrastructure/INTEGRATIONS]] — cómo se habla con Ollama y cómo se listan los modelos.
- [[backend/DECISIONS]] — decisiones de configuración aún abiertas.
- [[PROJECT]] — requisitos en la máquina y despliegue.

---
description: Resolver problemas encontrados durante el desarrollo: analiza, propone soluciones y ejecuta.
---

# Comando: Resolver

**Entrada del usuario:** `$ARGUMENTS`

Eres un agente de resolución de problemas. Tu objetivo es identificar la causa raíz de un problema, proponer una solución clara y ejecutarla de forma segura.

---

## Flujo de trabajo

### Paso 1: Analizar el problema

1. Lee y comprende la entrada del usuario (`$ARGUMENTS`).
2. Si el usuario menciona errores de consola, ejecuta el comando relevante para capturar los errores actuales.
3. Busca en el código fuente las áreas relacionadas con el error.
4. Identifica la causa raíz del problema.

### Paso 2: Proponer una solución

1. Describe el problema encontrado de forma concisa.
2. Propón una solución específica con los archivos a modificar y los cambios a realizar.
3. **Si la solución requiere actualizar documentación**, menciónalo explícitamente en la propuesta.
4. Espera aprobación del usuario antes de ejecutar.

### Paso 3: Ejecutar la solución

1. Una vez aprobada, implementa los cambios.
2. Verifica que el problema esté resuelto (ejecuta tests, linter, tipocheck, o el comando que aplique).
3. Si la solución requiere documentación actualizada, ejecuta esos cambios también.

---

## Reglas

1. **Nunca** asumas causas sin evidencia. Siempre respalda tu análisis con código o logs.
2. **Siempre** muestra la propuesta antes de ejecutar. El usuario debe aprobar.
3. **Siempre** verifica que el problema se resolvió después de ejecutar.
4. **Documentación**: Si la solución afecta el comportamiento del sistema, avisa que se necesita actualizar la documentación correspondiente.
5. Si el problema es ambiguo, pregunta al usuario para clarificar antes de proponer.
6. No ejecutes cambiosdestructivos sin confirmación explícita.

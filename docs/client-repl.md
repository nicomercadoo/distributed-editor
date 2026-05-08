# REPL del cliente

Este documento describe el REPL simple implementado en `internal/client/repl/`.
Su proposito es proveer una interfaz interactiva basica para conectar, enviar
comandos y mostrar estado, dejando la logica real para integrar despues.

## Alcance

- Parser de comandos con soporte de comillas y escapes.
- Loop principal con `readline` para prompt, historial y navegacion con tecla UP.
- Autocompletado por prefijo para comandos basicos.
- Stubs de handlers para conectar la logica real mas adelante.
- Salida limpia con `CTRL-C` y `CTRL-D`.

## Estructura

```text
internal/client/repl/
  command.go    // struct Command
  parser.go     // Parse() y splitFields()
  repl.go       // loop principal + handlers
  state.go      // estado local del REPL
```

## Comandos soportados (stubs)

- `connect <uri>`
- `disconnect`
- `send <op>`
- `show`
- `help`
- `exit`

## Prompt dinamico

- `>>>` por defecto
- `OK >` cuando el estado indica conexion activa
- `! >` si un handler devuelve error

## Autocompletado

Se usa `readline.NewPrefixCompleter` con:

- `connect`
- `disconnect`
- `send insert`
- `send delete`
- `show`
- `help`
- `exit`

## Dependencia

- `github.com/chzyer/readline`

## Integracion futura

- Reemplazar los stubs en `registerHandlers()` con llamadas reales.
- Persistir historial en un archivo propio del cliente.
- Parsear `send <op>` a estructuras de operacion reales.

# REPL del cliente

Este documento describe el REPL implementado en `internal/client/repl/`.
Provee una interfaz interactiva para conectar, enviar comandos y recibir
operaciones con transformacion local.

## Alcance

- Parser de comandos con soporte de comillas y escapes.
- Loop principal con `readline` para prompt, historial y navegacion con tecla UP.
- Autocompletado por prefijo para comandos basicos.
- Handlers con logica basica de conexion, envio y transformacion.
- Transformacion de operaciones entrantes contra colas locales.
- Salida limpia con `CTRL-C` y `CTRL-D`.

## Estructura

```text
internal/client/repl/
  command.go    // struct Command
  parser.go     // Parse() y splitFields()
  repl.go       // loop principal + handlers + listener
  state.go      // estado local del REPL + colas
```

## Comandos soportados

- `connect <uri>`
- `disconnect`
- `insert <char> <pos>`
- `delete <pos>`
- `show`
- `help`
- `exit`

## Prompt

- Se mantiene fijo en `> `.

## Autocompletado

Se usa `readline.NewPrefixCompleter` con:

- `connect`
- `disconnect`
- `insert`
- `delete`
- `show`
- `help`
- `exit`

## Dependencia

- `github.com/chzyer/readline`

## Integracion futura

- Persistir historial en un archivo propio del cliente.
- Conectar `show` con el estado local del documento.
- Mejorar el reporte de errores y validaciones.

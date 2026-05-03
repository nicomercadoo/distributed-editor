# Estructura de protocolos

Esta seccion resume la reorganizacion del paquete `internal/protocol` en tres
subpaquetes, separando documento, mensajes y operaciones.

## Nueva estructura

```text
internal/protocol/
  document/
    document_local_state.go
    document_server_state.go
    document_rep.go
  ack/
    ack.go
  message/
    message.go
  operation/
    operation.go
    operation_type.go
```

## Cambios de paquetes

- `internal/protocol/document` contiene:
  - `DocumentLocalState`
  - `DocumentServerState`
  - `DocumentRep`, `ID`
- `internal/protocol/ack` contiene:
  - `Ack`
- `internal/protocol/message` contiene:
  - `Message`
  - helpers `NewMessage`, `DecodeOperation`, `DecodeAck`
- `internal/protocol/operation` contiene:
  - `OperationType`, `Insert`, `Delete`
  - `Operation`

## Actualizacion de imports

Se actualizaron los imports y referencias para apuntar a los nuevos paquetes:

- `internal/server/hub.go`
- `internal/server/client.go`
- `internal/server/socket.go`
- `internal/client/ws_client.go`
```

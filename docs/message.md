# Mensajes en WebSocket

Este documento describe el wrapper `Message` y los cambios realizados para
soportar multiples tipos de mensajes por el mismo WebSocket.

## Cambios principales

- Se introdujo `Message` como wrapper con `type` y `payload` JSON.
- Se agrego `AckMessage` con el campo `revision`.
- Cliente y servidor ahora envian y reciben `Message` en ambos sentidos.

## Formato de mensaje

```json
{
  "type": "operation",
  "payload": {
    "op": "I",
    "pos": 0,
    "content": "a"
  }
}
```

```json
{
  "type": "ack",
  "payload": {
    "revision": 3
  }
}
```

## Impacto en el flujo

- El cliente envia `Message(type=operation)` en lugar de `OperationMessage`.
- El servidor responde con `Message(type=ack)` al emisor.
- El servidor retransmite `Message(type=operation)` a los demas clientes.

## Archivos afectados

- `internal/protocol/messages.go`
- `internal/server/client.go`
- `internal/server/hub.go`
- `internal/server/socket.go`
- `internal/client/ws_client.go`

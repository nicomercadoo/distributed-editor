# Resumen: Refactorización de Reintentos de Red y Purificación del SyncEngine

## 1. El Problema: Acoplamiento de Red en la Lógica de OT
Inicialmente, la estructura `InFlightOperation` (que contiene campos puramente de red como `attempts` y `received`) se encontraba integrada dentro del `SyncEngine`.
Esto violaba el Principio de Responsabilidad Única (SRP) y nuestra Arquitectura Híbrida porque:
- Mezclaba la lógica matemática de Transformación Operacional (OT) con la lógica de transporte (reintentos, timeouts).
- Obligaba al `SyncEngine` a tener que gestionar temporizadores, ensuciando su bucle rápido de procesamiento de eventos.

## 2. La Solución: Delegación a Actores de Red (CSP)
Decidimos que la responsabilidad de reintentar el envío de un paquete perdido o demorado debe recaer estrictamente sobre la capa de I/O (los actores de red).

Para lograr esto sin romper el modelo asíncrono (CSP):
1. Se movió `InFlightOperation` al dominio de la red (`net_actors.go`).
2. Se introdujo un **Canal Dedicado (`networkAcks`)** exclusivo para la comunicación interna entre el `Listener` y el `Sender`.

## 3. Cambios Implementados

1. **Purificación del `SyncEngine`:**
   - Volvió a almacenar simplemente un puntero a `operation.Operation`. Su única responsabilidad respecto a la operación "en vuelo" es utilizarla para transformar operaciones remotas entrantes hasta que llegue el `AckApplied`.

2. **El `Listener` como Demultiplexor:**
   - Actúa como un enrutador inteligente. Al leer el socket, hace un `switch` del tipo de mensaje. Si es una `Operation` o un `AckApplied`, los manda al canal entrante del motor. Si es un `AckReceived` preliminar, lo desvía inmediatamente al canal `networkAcks` hacia el `Sender`.

3. **El `Sender` como Gestor de Reintentos:**
   - Su bucle `select` ahora maneja un `time.Ticker`.
   - Cuando despacha una operación, arranca el temporizador.
   - Si recibe señal por `networkAcks`, cancela el temporizador (el paquete llegó al servidor).
   - Si expira el temporizador (`<-ticker.C`), resta un intento (`attempt`) y reenvía. Si se agotan los intentos, aborta.

4. **Estado de Red Inspeccionable (UI) con Concurrencia Encapsulada:**
   - La estructura `InFlightOperation` encapsula su propia concurrencia usando un `sync.RWMutex`.
   - El `ConnectionManager` mantiene un único puntero centralizado `inFlightOp *InFlightOperation` en lugar de variables sueltas.
   - El `Sender` modifica el estado interno usando `Lock()` (ej. `DecrementAttempt()`), mientras que la Web UI / REPL leen simultáneamente los valores a través del Manager con `RLock()` (ej. `Attempts()`), logrando seguridad frente a Data Races sin inflar el Manager con código redundante.

5. **Propagación de Errores Nativos en Goroutines:**
   - El método `writeToSocket` ahora devuelve un `error` en caso de fallar la escritura en el WebSocket.
   - En lugar de cerrar la conexión a bajo nivel, `Sender.Run` hace un `return err`.
   - Esto permite que la goroutine se detenga limpiamente y que el `ConnectionManager` padre capture el error, cierre la conexión y notifique a la UI en un solo lugar.

---

## 4. Diagramas de Arquitectura y Flujo

### A. Estructura y Flujo de Datos (Plano de Datos)

```mermaid
flowchart TB
    subgraph UI [AppClient / Inspector]
        UI_Control(Web UI / REPL) -- "Lee intentos\n(RLock)" --> CM
    end

    subgraph Capa Red [Capa de Red - net_actors]
        CM[ConnectionManager\nGuarda estado de red]
        Sender[Sender\nBucle con Ticker]
        Listener[Listener\nDemultiplexor]
    end

    subgraph Capa OT [Capa Algorítmica]
        SE[SyncEngine]
    end

    SE -- "1. Outbound (chan)" --> Sender
    Sender -- "Actualiza estado (Lock)" --> CM
    Sender == "2. Envia Operación" ==> SERVER((Servidor))
    
    SERVER == "3. Responde AckReceived" ==> Listener
    Listener -- "4. networkAcks (chan)" --> Sender
    
    SERVER == "5. Responde AckApplied" ==> Listener
    Listener -- "6. Incoming (chan)" --> SE
```

### B. Flujo de Ejecución (Secuencia de un Reintento)

```mermaid
sequenceDiagram
    participant SE as SyncEngine
    participant S as Sender
    participant L as Listener
    participant WS as Servidor

    SE-)S: Envía operación por canal Outbound
    activate S
    S->>S: Crea InFlightOperation (3 attempts)
    S->>WS: Escribe JSON (Operación)
    S->>S: Inicia Ticker (Ej: 500ms)
    
    Note over S, WS: Falla la red o hay latencia severa...
    
    S->>S: Ticker Expira (Timeout)
    S->>S: Decrementa Attempt (quedan 2)
    S->>WS: Reenvía JSON (Operación)
    
    Note over S, WS: La red se recupera...
    
    WS->>L: Llega AckReceived
    L-)S: Envía señal por canal networkAcks
    S->>S: Marca como Recibido = true
    S->>S: Apaga Ticker (Detiene reintentos)
    
    WS->>L: Llega AckApplied (definitivo)
    L-)SE: Envía evento por canal Incoming
    SE->>SE: Aplica revisión y procesa pending
    deactivate S
```

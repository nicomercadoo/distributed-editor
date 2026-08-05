# Diagramas de Arquitectura (Mermaid)

A continuación se detallan los diagramas solicitados para comprender la estructura interna del cliente, del servidor y cómo se comunican entre sí.

## 1. Estructura del Cliente

El cliente está desacoplado de la interfaz gráfica (funciona tanto para WASM en el navegador como para la terminal) a través de un orquestador principal llamado `AppClient`. Este orquestador coordina el motor de sincronización (`SyncEngine`) y el gestor de red (`ConnectionManager`).

```mermaid
graph TD
    subgraph ClientNode [Componentes del Cliente]
        UI[Interfaz de Usuario<br>WASM / CLI] -->|Llamadas| AC[AppClient]
        
        AC -->|Gestiona| SE[SyncEngine]
        AC -->|Gestiona| CM[ConnectionManager]
        
        SE -->|Transformaciones| OT[Módulo OT]
        
        subgraph EstadoLocal [Estado Interno]
            Doc[Documento String]
            Rev[Revisión Actual]
            Colas[Operaciones InFlight / Pending]
        end
        SE -.-> EstadoLocal
        
        CM -->|Inicia/Supervisa| WSConn[Conexión WebSocket]
        
        SE <-->|Canales Go:<br>LocalOps, Incoming, Outbound| CM
    end
```

## 2. Estructura del Servidor

El servidor está diseñado alrededor de un **Hub** concurrente que gestiona múltiples clientes WebSocket, validando cada operación recibida contra el estado único de la verdad mantenido por `DocumentServerState`.

```mermaid
graph TD
    subgraph ServerNode [Componentes del Servidor]
        Hub[Hub Central]
        
        Hub -->|Mantiene coleccion de| Clients[(Mapa de Clientes)]
        Hub -->|Protege Estado| DocState[DocumentServerState]
        
        DocState -.->|Texto| Doc[Documento Global]
        DocState -.->|Historial| RevLog[Revision Log]
        
        DocState -->|Valida y Desplaza Índices| OT[Módulo OT]
        
        subgraph ClienteConectado [Estructura por Cliente]
            CConn[Client Wrapper]
            CConn -->|Goroutine: readPump| HubDispatch[Canal: Dispatch]
            HubSend[Canal: Send] -->|Goroutine: writePump| CConn
        end
        
        Hub -->|Broadcast de Ops / ACK| HubSend
    end
```

## 3. Flujo de Conexión y Comunicación (Cliente - Servidor)

El siguiente diagrama de secuencia muestra el camino completo que recorre una operación desde que el usuario tipea una tecla hasta que el servidor la confirma, pasando opcionalmente por Toxiproxy para simular las condiciones de red adversas.

```mermaid
sequenceDiagram
    participant UI as Interfaz (WASM/CLI)
    participant SE as SyncEngine (Cliente)
    participant CM as ConnectionManager
    participant TP as Toxiproxy (Opcional)
    participant Hub as Servidor (Hub)
    
    %% Flujo de Operación Local
    UI->>SE: Operación Local (Ej: Insert)
    SE->>SE: Aplica cambio optimista en doc local
    SE->>CM: Pasa Operación al canal Outbound
    CM->>TP: Envía JSON sobre WebSocket
    TP->>Hub: Redirige tráfico (aplica delay si corresponde)
    
    %% Procesamiento del lado del servidor
    Hub->>Hub: readPump lee JSON y envía a Dispatch
    Hub->>Hub: DocumentServerState verifica Revisión
    Hub->>Hub: Si hay conflicto, usa OT contra el RevisionLog
    Hub->>Hub: Aplica cambio final y guarda en Log
    
    %% Respuesta
    Hub-->>TP: Envía ACK de Revisión al autor (writePump)
    Hub-->>TP: Broadcast de Op transformada al RESTO
    
    %% Manejo en el cliente original
    TP-->>CM: Retorna el ACK
    CM-->>SE: Canal Incoming recibe ACK
    SE->>SE: Saca Op de InFlight, incrementa Revisión local
    SE-->>UI: Dispara evento updateState()
```
### Flujo de Datos

```mermaid
flowchart TB
    subgraph Capa Ususario [Client]
        UI[Web UI / REPL]
    %% end
        subgraph Capa Red [Net Actors]
            %% CM[ConnectionManager]
            Sender[Sender]
            Listener[Listener]
        end

        subgraph Capa OT [Document]
            SE[SyncEngine]
        end
    end

    UI e1@-- "LocalOps" --> SE
    
    SE e2@-- "Outbound" --> Sender

    Sender e3@-- "Web Socket" --> Server((Server))
    Server e4@-- "WebSocket" --> Listener
    
    Listener e5@-- "NetworkAcks" --> Sender
    Listener e6@-- "Incoming" --> SE

    classDef animate stroke-dasharray: 9,5,stroke-dashoffset: 900,animation: dash 25s linear infinite;
    class e1 animate
    class e2 animate
    class e3 animate
    class e4 animate
    class e5 animate
    class e6 animate
```

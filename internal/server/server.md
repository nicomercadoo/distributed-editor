Lo que implementamos aquí es el patrón estándar de concurrencia en Go para manejar conexiones de red persistentes, conocido comúnmente como el patrón "Hub & Client". 

La arquitectura se basa en separar estrictamente las operaciones de I/O de red (lectura/escritura de sockets) de la gestión del estado en memoria (el documento y los usuarios conectados). Esto se logra utilizando goroutines independientes que se comunican exclusivamente a través de canales (*channels*), evitando así la necesidad de usar bloqueos manuales (*mutexes*) y previniendo *race conditions*.

Aquí está el desglose técnico de cada componente:

### 1. `internal/server/socket.go` (El Punto de Entrada)
Este archivo expone la interfaz HTTP. Su única responsabilidad es inicializar las conexiones.
* **Upgrade:** Toma la petición HTTP entrante y realiza el *handshake* del protocolo RFC 6455 para cambiar la conexión a un socket bidireccional de WebSocket.
* **Inicialización:** Crea una nueva instancia del `struct Client`, pasándole la referencia de la conexión recién creada y un puntero al `GlobalHub`. Además, inicializa el canal `send` de ese cliente específico.
* **Registro y Ejecución:** Envía el nuevo puntero del cliente al canal `Register` del Hub y lanza las funciones `ReadPump` y `WritePump` como dos goroutines concurrentes. Luego de esto, el hilo principal de la función HTTP termina, pero la conexión se mantiene viva en fondo a través de las goroutines.

### 2. `internal/server/client.go` (La Interfaz de Red)
El cliente actúa como un adaptador entre el socket TCP de red subyacente y el Hub en memoria. Se divide en dos bucles infinitos que corren en hilos separados:
* **`ReadPump`:** Su hilo se bloquea esperando datos desde la red mediante `conn.ReadJSON()`. Apenas recibe y deserializa un JSON válido en una estructura `protocol.Operation`, lo inyecta en el canal `Broadcast` del Hub. Si la red falla o el usuario cierra la conexión, rompe el bucle y envía su propia referencia al canal `Unregister` del Hub.
* **`WritePump`:** Su hilo se bloquea esperando datos desde la memoria, específicamente desde su canal interno `c.send`. Cuando el Hub deposita una operación en este canal, el bucle despierta, serializa la estructura a JSON y la escribe en el socket mediante `conn.WriteJSON()`.

### 3. `internal/server/hub.go` (El Gestor de Estado)
Es el núcleo del servidor. Mantiene la "fuente de la verdad" en memoria: el `DocumentLog` (la secuencia de operaciones) y el mapa de clientes activos. Se ejecuta en una única goroutine iniciada en el `main`.
* **Serialización de eventos:** La función `Run()` contiene un bucle `for` infinito con una sentencia `select`. Esto es crítico: garantiza que el Hub procese exactamente un evento por vez, ya sea un registro, un desregistro o un broadcast. Esto hace que las operaciones de escritura en el mapa `clients` y en el slice `DocumentLog` sean 100% seguras en entornos concurrentes.
* **Flujo de Registro:** Cuando recibe un cliente por el canal `Register`, lo añade al mapa y luego itera secuencialmente sobre el `DocumentLog`, depositando todo el historial en el canal `send` de ese cliente para cumplir con el requerimiento de enviar la última versión al conectar.
* **Flujo de Broadcast:** Cuando recibe una operación por el canal `Broadcast`, primero la hace persistente agregándola al slice `DocumentLog`. Luego, itera sobre el mapa de clientes activos y deposita esa misma operación en el canal `send` de cada uno de ellos (lo que despertará a sus respectivos `WritePumps`).

### Resumen del ciclo de vida de un mensaje
Si un usuario envía un comando desde su terminal:
1. El socket del servidor recibe el JSON.
2. El `ReadPump` del cliente "A" lo deserializa y lo empuja al canal `Broadcast`.
3. El `Hub` procesa el `Broadcast`, lo guarda en el Log, y lo empuja al canal `send` de todos los clientes en el mapa (incluyendo al cliente "B").
4. El `WritePump` del cliente "B" detecta el mensaje en su canal `send`, lo serializa a JSON y lo escupe por la red hacia el frontend.

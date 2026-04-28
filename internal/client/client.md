El propósito central es establecer la conexión de red y luego dividir la ejecución de tu terminal en dos flujos concurrentes: uno para escuchar la red y otro para escuchar el teclado.

Aquí está el desglose de cómo opera bajo el capó:

### 1. Establecimiento de la Conexión
La función `StartSession` comienza ejecutando `websocket.DefaultDialer.Dial(url, nil)`.
* Esto intenta resolver el *handshake* HTTP con el servidor en la URL proporcionada.
* Si es exitoso, devuelve un objeto de conexión (`conn`) que se utilizará tanto para leer como para escribir.
* La instrucción `defer conn.Close()` garantiza que, sin importar cómo termine la ejecución (incluso si hay un error crítico), el socket se cerrará de forma segura y se liberarán los recursos.

### 2. Goroutine de Lectura (Escuchar al Servidor)
Inmediatamente después de conectar, se lanza un subproceso con `go func() { ... }()`.
* Este bloque entra en un bucle infinito `for {}` cuya única tarea es esperar datos de la red.
* La línea `conn.ReadJSON(&op)` bloquea la ejecución de esta *goroutine* específica hasta que llega un mensaje desde el servidor. Cuando llega, lo deserializa directamente en la estructura compartida `protocol.Operation`.
* Si la conexión se corta (por ejemplo, si detienes el servidor), el error no es nulo, se imprime un aviso en consola y la ejecución del programa termina por completo mediante `os.Exit(0)`.
* Este diseño concurrente permite que el cliente reciba las notificaciones en el exacto momento en que el servidor las distribuye, operando de forma totalmente independiente a lo que estés haciendo en tu línea de comandos.

### 3. Bucle Principal (Escuchar al Teclado)
El hilo principal del programa continúa hacia la configuración del `bufio.NewScanner(os.Stdin)`.
* La ejecución se detiene en `scanner.Scan()`. Solo avanza cuando presionas "Enter" en la terminal.
* Una vez que capturas el texto ingresado, el programa empaqueta esa información en un `struct` del tipo `protocol.Operation`. En la versión actual, está estructurado de manera rígida con valores fijos (tipo "insert" en la posición 0) para realizar una prueba de comunicación pura.
* Luego, `conn.WriteJSON(op)` se encarga de serializar el `struct` en formato JSON y enviarlo por la red hacia el servidor.
* Al ejecutarse en el hilo principal, garantiza que cuando se integre el parser de la librería Cobra que está implementando tu compañero, la captura de la entrada estándar y el ruteo de los comandos no se superpongan ni interrumpan la lectura de los eventos de red.
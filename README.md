# Editor Distribuido

Implementación de un sistema de edición colaborativo utilizando **Operational Transformation (OT)**, con un servidor central y múltiples clientes soportados tanto desde la terminal (REPL) como desde una interfaz web, con la capacidad de simular problemas de red a través de Toxiproxy.

### Developed by
- [Nicolás Mercado](https://github.com/nicomercadoo)
- [Santiago Sosa Moressi](https://github.com/santisosa9)

### Arquitectura e Implementación
Para más información sobre la arquitectura e implementación del proyecto lee [PROJECT.md](./PROJECT.md)

## Dependencias

Para poder levantar todos los servicios del proyecto es necesario contar con:

- **Go (Golang)**: Para compilar y ejecutar el servidor, el REPL y el módulo WebAssembly.
- **Python 3**: Para servir los archivos de la interfaz web mediante `http.server`.
- **Docker**: Para levantar el contenedor de **Toxiproxy** que simula las condiciones y fallas de la red.

## Cómo Correr el Proyecto

### TL;DR 

El proyecto incluye un script orquestador (`run.sh`) que levanta todos los componentes en conjunto:
```bash
./run.sh all
```

O si se quiere borrar el historial del servidor:
```bash
./run.sh all -c
```

Esto iniciará en segundo plano (modo silencioso) el **Toxiproxy**, el **Servidor Central** y el **Servidor Web**, y dejará en primer plano el **REPL interactivo** para que comiences a usarlo de forma inmediata. Al presionar `Ctrl+C`, el script se encarga de limpiar y cerrar automáticamente todos los procesos que había levantado.

### Opciones de `run.sh`

El script permite iniciar componentes específicos. Se le pueden pasar múltiples comandos a la vez.
- **`toxiproxy`**: Inicia el simulador de red en Docker.
- **`server`**: Inicia el servidor central de OT.
- **`web`**: Inicia el servidor estático para el frontend.
- **`repl`**: Inicia el cliente interactivo en la terminal.
- **`all`**: Inicia todos los componentes mencionados.

Se puede usar la flag `-q` o `--quiet` al principio para ocultar los logs de los procesos en segundo plano:
```bash
./run.sh -q server web repl
```
O la flag `-l`o `--logs` para que redirigir el output de cada proceso a un archivo temporal:
```bash
./run.sh -l server web repl
```

### Ejecución Manual (Paso a Paso)

Para levantar cada componente con parámetros particulares:

1. **Toxiproxy**:
   Levanta el contenedor montando la configuración inicial desde el directorio actual (no carga los toxics de la configuración):
   ```bash
   docker run -d --rm --name toxiproxy --network host \
       -v "$(pwd)/toxiproxy.json:/toxiproxy.json" \
       ghcr.io/shopify/toxiproxy -config /toxiproxy.json
   ```

2. **Servidor Central**:
   ```bash
   go run cli/server/main.go start
   ```

3. **Cliente Web**:
   Inicia el servidor HTTP estático en la carpeta de la UI para poder acceder desde el navegador:
   ```bash
   python3 -m http.server 3000 --directory web_ui
   ```

4. **Cliente Terminal (REPL)**:
   ```bash
   go run cli/client/main.go start
   ```

## Interactuando con Toxiproxy vía `curl`

Toxiproxy expone una API HTTP de administración en el puerto **8474**. Nuestra configuración (`toxiproxy.json`) expone un proxy llamado `redes_proxy` en el puerto `8081` que redirige el tráfico al servidor real (que por defecto corre en `8080`).

**Ver el estado de todos los proxies activos**:
```bash
curl -s localhost:8474/proxies
```

**Ver las fallas ("toxics") aplicadas a `redes_proxy`**:
```bash
curl -s localhost:8474/proxies/redes_proxy/toxics
```

**Modificar la latencia dinámicamente**:
El proyecto viene configurado de fábrica con un toxic llamado `latencia`. Podés cambiarle el valor (en milisegundos) en tiempo real enviando un POST a la API:
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"attributes": {"latency": 1000}}' \
  localhost:8474/proxies/redes_proxy/toxics/latencia
```

**Eliminar la latencia (simular conexión perfecta)**:
```bash
curl -X DELETE localhost:8474/proxies/redes_proxy/toxics/latencia
```

**Volver a agregar la latencia (ej. 2 segundos con 500ms de jitter/variación)**:
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"name": "latencia", "type": "latency", "stream": "downstream", "attributes": {"latency": 2000, "jitter": 500}}' \
  localhost:8474/proxies/redes_proxy/toxics
```

**Simular una caída de la red**:
```bash
curl -s -X POST \
    -d '{ "enabled": false}' \
    http://localhost:8474/proxies/redes_proxy
```

**Volver a levantar la conexión**:
```bash
curl -s -X POST \
    -d '{ "enabled": true}' \
    http://localhost:8474/proxies/redes_proxy
```

Para mas ejemplos ver la documentación de [Toxiproxiy](https://github.com/shopify/toxiproxy#usage)

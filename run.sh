#!/bin/bash

cd "$(dirname "$0")"

QUIET=0
LOG_OUTPUT=0
while [[ "$1" == "-"* ]]; do
    if [[ "$1" == "-q" || "$1" == "--quiet" ]]; then
        QUIET=1
    elif [[ "$1" == "-l" || "$1" == "--logs" ]]; then
        LOG_OUTPUT=1
    else
        echo "Opción desconocida: $1"
        exit 1
    fi
    shift
done

if [ $# -eq 0 ]; then
    echo "Uso: ./run.sh [-q | -l] [comando1] [args1...] [comando2] [args2...] ..."
    echo ""
    echo "Opciones:"
    echo "  -q, --quiet Oculta los logs de los procesos en segundo plano."
    echo "  -l, --logs  Guarda los logs de los procesos en background en archivos temporales."
    echo ""
    echo "Comandos disponibles:"
    echo "  toxiproxy   Inicia Toxiproxy en Docker."
    echo "  server      Inicia el Servidor central."
    echo "  web         Inicia el Servidor para los Clientes Web."
    echo "  repl        Inicia un Cliente REPL interactivo."
    echo "  all         Equivale a: toxiproxy, server, web y repl. Sin logs."
    echo ""
    echo "Ejemplo múltiple:"
    echo "  ./run.sh server --port 8081 web -p 3002 web -p 3003"
    exit 0
fi

# Guardamos los argumentos restantes en un arreglo para poder iterarlos y
# extraer los parámetros específicos que corresponden a cada comando.
ARGS=("$@")

# Buscamos en qué posiciones (índices) del arreglo ARGS están los nombres de los comandos
# válidos (toxiproxy, server, web, repl, all). Esto nos permite partir los argumentos en bloques.
COMMAND_INDEXES=()
for i in "${!ARGS[@]}"; do
    arg="${ARGS[$i]}"
    case "$arg" in
        toxiproxy|server|web|repl|all)
            COMMAND_INDEXES+=($i)
            ;;
    esac
done

if [ ${#COMMAND_INDEXES[@]} -eq 0 ]; then
    echo "Error: No se encontró ningún comando válido (toxiproxy, server, web, repl, all)."
    exit 1
fi

if [ ${COMMAND_INDEXES[0]} -ne 0 ]; then
    echo "Error: Se pasaron argumentos antes de ningún comando. Asegurate de poner el nombre del comando primero."
    exit 1
fi

COMMAND_INDEXES+=(${#ARGS[@]}) # Añadimos el final para poder calcular los slices de argumentos fácilmente

PIDS=""
HAS_TOXI=0

# Función de limpieza que se ejecuta automáticamente al salir (ej. con Ctrl+C)
cleanup() {
    if [ -n "$PIDS" ] || [ $HAS_TOXI -eq 1 ]; then
        echo -e "\n[!] Limpiando servicios..."
        if [ -n "$PIDS" ]; then
            for pid in $PIDS; do
                # IMPORTANTE: Usamos 'pkill -P' para matar primero a los procesos hijos.
                # Esto es crucial porque 'go run' compila un binario temporal y lo ejecuta
                # como un proceso hijo. Si solo matamos el padre, el servidor compilado
                # queda corriendo de fondo (zombie) ocupando el puerto.
                pkill -P $pid 2>/dev/null
                kill $pid 2>/dev/null
            done
        fi
        if [ $HAS_TOXI -eq 1 ]; then
            docker stop toxiproxy >/dev/null 2>&1
        fi
    fi
}
trap cleanup EXIT # Registramos el cleanup para señales de salida y Ctrl+C

# Lanza un proceso en segundo plano (background) y guarda su PID para la limpieza posterior.
run_background() {
    local name="$1"
    shift

    if [ $LOG_OUTPUT -eq 1 ]; then
        local log_file=$(mktemp /tmp/${name}_XXXXXX.log)
        echo "    [Log file: $log_file]"
        "$@" >"$log_file" 2>&1 &
    elif [ $QUIET -eq 1 ]; then
        "$@" >/dev/null 2>&1 &
    else
        "$@" &
    fi
    PIDS="$PIDS $!" # Concatenamos el PID del último comando ejecutado en background
}

start_toxiproxy() {
    echo "[+] Iniciando Toxiproxy en Docker..."
    docker rm -f toxiproxy >/dev/null 2>&1
    HAS_TOXI=1
    if [ $LOG_OUTPUT -eq 1 ]; then
        local log_file=$(mktemp /tmp/toxiproxy_XXXXXX.log)
        echo "    [Log file: $log_file]"
        docker run -d --rm --name toxiproxy --network host \
            -v "$(pwd)/toxiproxy.json:/toxiproxy.json" \
            ghcr.io/shopify/toxiproxy -config /toxiproxy.json "$@" >"$log_file" 2>&1
    elif [ $QUIET -eq 1 ]; then
        docker run -d --rm --name toxiproxy --network host \
            -v "$(pwd)/toxiproxy.json:/toxiproxy.json" \
            ghcr.io/shopify/toxiproxy -config /toxiproxy.json "$@" >/dev/null 2>&1
    else
        docker run -d --rm --name toxiproxy --network host \
            -v "$(pwd)/toxiproxy.json:/toxiproxy.json" \
            ghcr.io/shopify/toxiproxy -config /toxiproxy.json "$@"
    fi

    # Esperamos a que Toxiproxy levante y cargamos los toxics dinámicamente
    # ya que por diseño no lo hace.
    sleep 1
    if [ $LOG_OUTPUT -eq 1 ]; then
        python3 load_toxics.py >>"$log_file" 2>&1
    elif [ $QUIET -eq 1 ]; then
        python3 load_toxics.py >/dev/null 2>&1
    else
        python3 load_toxics.py
    fi
}

# Inicia el servidor central. Recibe 'is_last' como primer parámetro para saber
# si debe correr en primer plano (foreground) o segundo plano (background).
start_server() {
    local is_last=$1
    shift # Consumimos 'is_last', dejando el resto de los argumentos intactos ("$@")
    echo "[+] Iniciando Servidor..."
    if [ $is_last -eq 0 ]; then
        run_background server go run cli/server/main.go start "$@"
    else
        if [ $LOG_OUTPUT -eq 1 ]; then
            local log_file=$(mktemp /tmp/server_XXXXXX.log)
            echo "    [Log file: $log_file]"
            go run cli/server/main.go start "$@" >"$log_file" 2>&1
        elif [ $QUIET -eq 1 ]; then
            go run cli/server/main.go start "$@" >/dev/null 2>&1
        else
            go run cli/server/main.go start "$@"
        fi
    fi
}

start_web() {
    local is_last=$1
    shift
    echo "[+] Iniciando Cliente Web..."
    if [ $is_last -eq 0 ]; then
        run_background web python3 -m http.server 3000 --directory web_ui "$@"
    else
        if [ $LOG_OUTPUT -eq 1 ]; then
            local log_file=$(mktemp /tmp/web_XXXXXX.log)
            echo "    [Log file: $log_file]"
            python3 -m http.server 3000 --directory web_ui "$@" >"$log_file" 2>&1
        elif [ $QUIET -eq 1 ]; then
            python3 -m http.server 3000 --directory web_ui "$@" >/dev/null 2>&1
        else
            python3 -m http.server 3000 --directory web_ui "$@"
        fi
    fi
}

start_repl() {
    local is_last=$1
    shift
    echo "[+] Iniciando REPL interactivo..."
    if [ $is_last -eq 0 ]; then
        # Si no es el último, lo forzamos al background igual, aunque el REPL interactivo no suele servir en bg
        run_background repl go run cli/client/main.go start "$@"
    else
        go run cli/client/main.go start "$@"
    fi
}

# --- LÓGICA DE EJECUCIÓN (Procesando cada comando y sus argumentos) ---

NUM_TASKS=$((${#COMMAND_INDEXES[@]} - 1))

for (( t=0; t<NUM_TASKS; t++ )); do
    start_idx=${COMMAND_INDEXES[$t]}
    end_idx=${COMMAND_INDEXES[$((t+1))]}

    cmd="${ARGS[$start_idx]}"

    # Extraemos los argumentos de ESTE comando específico (entre este índice y el siguiente comando)
    cmd_args=("${ARGS[@]:$((start_idx+1)):$((end_idx-start_idx-1))}")

    # Verificamos si es el último comando en la lista
    is_last=0
    if [ $((t+1)) -eq $NUM_TASKS ]; then
        is_last=1
    fi

    case "$cmd" in
        toxiproxy)
            start_toxiproxy "${cmd_args[@]}"
            ;;
        server)
            start_server $is_last "${cmd_args[@]}"
            ;;
        web)
            start_web $is_last "${cmd_args[@]}"
            ;;
        repl)
            start_repl $is_last "${cmd_args[@]}"
            ;;
        all)
            echo "[+] Iniciando todos los componentes..."
            # Forzamos temporalmente la flag LOG_OUTPUT para que toxiproxy, server y web
            # logueen en archivos y no ensucien la consola, dejando que el REPL sea el visible.
            ORIG_LOG=$LOG_OUTPUT
            LOG_OUTPUT=1
            start_toxiproxy
            start_server 0
            start_web 0
            LOG_OUTPUT=$ORIG_LOG

            sleep 2 # Esperamos a que los servidores levanten antes de iniciar el REPL
            start_repl $is_last
            ;;
    esac
done

# Si el último comando fue toxiproxy, se corrió en Docker (desatachado).
# Necesitamos bloquear el script para que no termine inmediatamente y lance el cleanup.
last_cmd="${ARGS[${COMMAND_INDEXES[$((NUM_TASKS-1))]}]}"
if [ "$last_cmd" = "toxiproxy" ]; then
    echo "====================================================="
    echo " [Servicios corriendo. Presiona CTRL+C para salir]"
    echo "====================================================="
    if [ -n "$PIDS" ]; then
        wait $PIDS 2>/dev/null
    else
        sleep infinity
    fi
fi

#!/bin/bash

cd "$(dirname "$0")"

QUIET=0
if [[ "$1" == "-q" || "$1" == "--quiet" ]]; then
    QUIET=1
    shift
fi

if [ $# -eq 0 ]; then
    echo "Uso: ./run.sh [-q | --quiet] [comando1] [args1...] [comando2] [args2...] ..."
    echo ""
    echo "Opciones:"
    echo "  -q, --quiet Oculta los logs de los procesos en segundo plano."
    echo ""
    echo "Comandos disponibles:"
    echo "  toxiproxy   Inicia Toxiproxy en Docker."
    echo "  server      Inicia el Servidor central."
    echo "  web         Inicia el Cliente Web."
    echo "  repl        Inicia el Cliente REPL interactivo."
    echo "  all         Equivale a: toxiproxy, server, web y repl."
    echo ""
    echo "Ejemplo múltiple:"
    echo "  ./run.sh server --port 8081 web -p 3002 web -p 3003"
    exit 0
fi

# Guardamos los argumentos restantes
ARGS=("$@")

# Buscamos en qué posiciones (índices) están los nombres de los comandos
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

cleanup() {
    if [ -n "$PIDS" ] || [ $HAS_TOXI -eq 1 ]; then
        echo -e "\n[!] Limpiando servicios..."
        if [ -n "$PIDS" ]; then
            kill $PIDS 2>/dev/null
        fi
        if [ $HAS_TOXI -eq 1 ]; then
            docker stop toxiproxy >/dev/null 2>&1
        fi
    fi
}
trap cleanup EXIT

run_background() {
    if [ $QUIET -eq 1 ]; then
        "$@" >/dev/null 2>&1 &
    else
        "$@" &
    fi
    PIDS="$PIDS $!"
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
            echo "[+] Iniciando Toxiproxy en Docker..."
            docker rm -f toxiproxy >/dev/null 2>&1
            HAS_TOXI=1
            if [ $QUIET -eq 1 ]; then
                docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy "${cmd_args[@]}" >/dev/null 2>&1
            else
                docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy "${cmd_args[@]}"
            fi
            ;;
        server)
            echo "[+] Iniciando Servidor..."
            if [ $is_last -eq 0 ]; then
                run_background go run cli/server/main.go start "${cmd_args[@]}"
            else
                if [ $QUIET -eq 1 ]; then
                    go run cli/server/main.go start "${cmd_args[@]}" >/dev/null 2>&1
                else
                    go run cli/server/main.go start "${cmd_args[@]}"
                fi
            fi
            ;;
        web)
            echo "[+] Iniciando Cliente Web..."
            if [ $is_last -eq 0 ]; then
                run_background go run cli/client/main.go start-web "${cmd_args[@]}"
            else
                if [ $QUIET -eq 1 ]; then
                    go run cli/client/main.go start-web "${cmd_args[@]}" >/dev/null 2>&1
                else
                    go run cli/client/main.go start-web "${cmd_args[@]}"
                fi
            fi
            ;;
        repl)
            echo "[+] Iniciando REPL interactivo..."
            if [ $is_last -eq 0 ]; then
                # Si no es el último, lo forzamos al background igual, aunque el REPL interactivo no suele servir en bg
                run_background go run cli/client/main.go start "${cmd_args[@]}"
            else
                go run cli/client/main.go start "${cmd_args[@]}"
            fi
            ;;
        all)
            echo "[+] Iniciando Toxiproxy, Server, Web y REPL..."
            docker rm -f toxiproxy >/dev/null 2>&1
            HAS_TOXI=1
            if [ $QUIET -eq 1 ]; then
                docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy >/dev/null 2>&1
            else
                docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy
            fi
            
            run_background go run cli/server/main.go start
            run_background go run cli/client/main.go start-web
            sleep 2
            
            if [ $is_last -eq 0 ]; then
                run_background go run cli/client/main.go start
            else
                go run cli/client/main.go start
            fi
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

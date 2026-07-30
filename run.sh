#!/bin/bash

# Nos aseguramos de estar en la raíz del proyecto
cd "$(dirname "$0")"

QUIET=0
if [[ "$1" == "-q" || "$1" == "--quiet" ]]; then
    QUIET=1
    shift
fi

COMMAND=$1

# Función auxiliar para ejecutar en segundo plano silenciando o no la salida
run_background() {
    if [ $QUIET -eq 1 ]; then
        "$@" >/dev/null 2>&1 &
    else
        "$@" &
    fi
}

case "$COMMAND" in
    toxiproxy)
        echo "[+] Iniciando Toxiproxy en Docker..."
        docker rm -f toxiproxy 2>/dev/null
        docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy
        ;;
    server)
        echo "[+] Iniciando Servidor..."
        if [ $QUIET -eq 1 ]; then
            go run ./cli/server >/dev/null 2>&1
        else
            go run ./cli/server
        fi
        ;;
    web)
        echo "[+] Iniciando Cliente Web..."
        if [ $QUIET -eq 1 ]; then
            go run ./cli/client start-web >/dev/null 2>&1
        else
            go run ./cli/client start-web
        fi
        ;;
    repl)
        echo "[+] Iniciando Cliente REPL..."
        go run ./cli/client start
        ;;
    all)
        echo "[+] Iniciando Toxiproxy..."
        docker rm -f toxiproxy 2>/dev/null
        if [ $QUIET -eq 1 ]; then
            docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy >/dev/null 2>&1
        else
            docker run -d --rm --name toxiproxy --network host ghcr.io/shopify/toxiproxy
        fi

        echo "[+] Iniciando Servidor..."
        run_background go run ./cli/server
        SERVER_PID=$!

        echo "[+] Iniciando Cliente Web..."
        run_background go run ./cli/client start-web
        WEB_PID=$!

        sleep 2
        echo "====================================================="
        echo " Servidor corriendo en puerto 8080"
        echo " Cliente Web corriendo en http://localhost:3000"
        if [ $QUIET -eq 1 ]; then
            echo " [Modo silencioso activado: los logs están ocultos]"
        fi
        echo "====================================================="
        echo "[+] Iniciando REPL interactivo..."
        
        # Atrapar CTRL+C o la salida del REPL para limpiar los procesos en background
        trap "echo -e '\n[!] Limpiando procesos de fondo (Servidor y Web) y Toxiproxy...'; kill $SERVER_PID $WEB_PID 2>/dev/null; docker stop toxiproxy 2>/dev/null" EXIT

        go run ./cli/client start
        ;;
    *)
        echo "Uso: ./run.sh [-q | --quiet] [comando]"
        echo ""
        echo "Opciones:"
        echo "  -q, --quiet Oculta los logs de los procesos ejecutados (redirige a /dev/null)."
        echo ""
        echo "Comandos disponibles:"
        echo "  toxiproxy   Inicia únicamente Toxiproxy en Docker."
        echo "  server      Inicia únicamente el Servidor central."
        echo "  web         Inicia únicamente el Cliente Web y su puente local."
        echo "  repl        Inicia únicamente el Cliente REPL interactivo."
        echo "  all         Inicia Toxiproxy, el Servidor y la Web en segundo plano,"
        echo "              y abre el REPL interactivo aquí. (Al salir, apaga todo)."
        ;;
esac

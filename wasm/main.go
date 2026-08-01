//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"
	"time"

	"proyecto-final-redes/internal/client"
	"proyecto-final-redes/internal/protocol/message"
)

func main() {
	c := make(chan struct{}, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localOps := make(chan message.Operation, 100)
	incoming := make(chan message.ServerEvent, 100)
	outbound := make(chan message.Operation, 100)

	notifications := make(chan string, 100)

	syncEngine := client.NewSyncEngine(localOps, incoming, outbound, notifications)
	connManager := client.NewConnectionManager(ctx, incoming, outbound, notifications, syncEngine.GetRevision)
	appClient := client.NewAppClient(syncEngine, connManager)

	go syncEngine.Run(ctx)

	// Iniciar un loop que envíe el estado a JS periódicamente
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
		defer ticker.Stop()

		var lastDoc string
		var lastRev = -1
		var lastPending = -1
		var lastInFlight bool
		var lastConnected bool

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				doc := appClient.GetDocument()
				rev := appClient.GetRevision()
				pending := appClient.GetPendingCount()
				inFlight := appClient.HasInFlight()
				connected := appClient.IsConnected()

				if doc != lastDoc || rev != lastRev || pending != lastPending || inFlight != lastInFlight || connected != lastConnected {
					lastDoc = doc
					lastRev = rev
					lastPending = pending
					lastInFlight = inFlight
					lastConnected = connected

					updateFn := js.Global().Get("updateState")
					if !updateFn.IsUndefined() {
						state := map[string]interface{}{
							"document":     doc,
							"revision":     rev,
							"pendingCount": pending,
							"hasInFlight":  inFlight,
							"isConnected":  connected,
						}
						// Convert state to JS Object
						b, _ := json.Marshal(state)
						updateFn.Invoke(string(b))
					}
				}
			}
		}
	}()
	// Leer notificaciones y hacer console.log
	go func() {
		for notif := range notifications {
			js.Global().Get("console").Call("log", "WASM:", notif)
		}
	}()

	// Exponer funciones a JS
	js.Global().Set("connectWasm", js.FuncOf(func(this js.Value, args []js.Value) any {
		uri := args[0].String()
		go func() {
			if err := appClient.Connect(uri); err != nil {
				js.Global().Get("console").Call("error", "Error connect: ", err.Error())
			}
		}()
		return nil
	}))

	js.Global().Set("disconnectWasm", js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() {
			appClient.Disconnect()
		}()
		return nil
	}))

	js.Global().Set("insertWasm", js.FuncOf(func(this js.Value, args []js.Value) any {
		pos := args[0].Int()
		char := args[1].String()
		localOps <- message.Operation{
			Op:       message.Insert,
			Position: pos,
			Content:  char,
			Revision: appClient.GetRevision(),
		}
		return nil
	}))

	js.Global().Set("deleteWasm", js.FuncOf(func(this js.Value, args []js.Value) any {
		pos := args[0].Int()
		localOps <- message.Operation{
			Op:       message.Delete,
			Position: pos,
			Revision: appClient.GetRevision(),
		}
		return nil
	}))

	js.Global().Set("setLatencyWasm", js.FuncOf(func(this js.Value, args []js.Value) any {
		durationStr := args[0].String()
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			js.Global().Get("console").Call("error", "Invalid latency:", err.Error())
			return nil
		}
		go func() {
			if err := appClient.SetLatency(d); err != nil {
				js.Global().Get("console").Call("error", "Error setting latency:", err.Error())
			}
		}()
		return nil
	}))

	<-c
}

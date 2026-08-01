//go:build !js || !wasm

package client

import (
	"fmt"
	"net/url"
	"time"
	"errors"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/gorilla/websocket"
)

func (m *ConnectionManager) dial(uri string) (WSConn, error) {
	// Setup Toxiproxy if available
	proxyURI := uri
	u, err := url.Parse(uri)
	if err == nil {
		toxiClient := toxiproxy.NewClient("localhost:8474")
		proxyName := fmt.Sprintf("redes_proxy_%d", time.Now().UnixNano())

		p, errProxy := toxiClient.CreateProxy(proxyName, "localhost:0", u.Host)
		if errProxy == nil {
			m.toxiProxy = p
			u.Host = p.Listen
			proxyURI = u.String()
		} else {
			select {
			case m.Notifications <- "Aviso: Toxiproxy no está corriendo. Conectando directo a " + u.Host:
			default:
			}
		}
	}

	c, _, err := websocket.DefaultDialer.Dial(proxyURI, nil)
	if err != nil {
		return nil, fmt.Errorf("Error al conectar: %w", err)
	}

	return c, nil
}

func (m *ConnectionManager) cleanupProxy() {
	if m.toxiProxy != nil {
		m.toxiProxy.Delete()
		m.toxiProxy = nil
	}
}

func (m *ConnectionManager) setLatencyNative(d time.Duration) error {
	if m.toxiProxy == nil && d > 0 {
		return errors.New("toxiproxy no está activo; no se puede aplicar latencia")
	}

	if m.toxiProxy != nil {
		latencyMs := d.Milliseconds()
		
		if d > 0 {
			_, err := m.toxiProxy.AddToxic("latency_toxic", "latency", "downstream", 1.0, toxiproxy.Attributes{
				"latency": latencyMs,
			})
			if err != nil {
				_, err = m.toxiProxy.UpdateToxic("latency_toxic", 1.0, toxiproxy.Attributes{
					"latency": latencyMs,
				})
				if err != nil {
					return fmt.Errorf("no se pudo configurar la latencia en toxiproxy: %w", err)
				}
			}
		} else {
			err := m.toxiProxy.RemoveToxic("latency_toxic")
			if err != nil && err.Error() != "not found" {
				// We can ignore error if it was not found
			}
		}
	}

	return nil
}

package client

// ConnectionController gestiona el ciclo de vida de la red
type ConnectionController interface {
	Connect(uri string) error
	Disconnect() error
	IsConnected() bool
}

// DocumentInspector permite lecturas de solo consulta sobre el estado de sincronización
type DocumentInspector interface {
	GetDocument() string
	GetRevision() int
	GetPendingCount() int
	HasInFlight() bool
	GetInFlightAttempts() int
	IsInFlightReceived() bool
}

// AppController es la fachada unificada que consumirá el REPL
type AppController interface {
	ConnectionController
	DocumentInspector
}

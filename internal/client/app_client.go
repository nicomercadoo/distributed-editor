package client

// AppClient es el struct principal del cliente
// Implementa la interfaz AppController
type AppClient struct {
	*SyncEngine
	*ConnectionManager
}

func NewAppClient(se *SyncEngine, cm *ConnectionManager) *AppClient {
	return &AppClient{
		SyncEngine:        se,
		ConnectionManager: cm,
	}
}

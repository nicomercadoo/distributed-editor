package client
import "proyecto-final-redes/internal/client/syncengine"

// AppClient is the main client struct that manages synchronization with the server.
// It implements the AppControler interface.
type AppClient struct {
	*syncengine.SyncEngine
	*ConnectionManager
}

func NewAppClient(se *syncengine.SyncEngine, cm *ConnectionManager) *AppClient {
	return &AppClient{
		SyncEngine:        se,
		ConnectionManager: cm,
	}
}

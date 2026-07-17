package client


// AppClient is the main client struct that manages synchronization with the server.
// It implements the AppControler interface.
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

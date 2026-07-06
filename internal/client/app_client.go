package client

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

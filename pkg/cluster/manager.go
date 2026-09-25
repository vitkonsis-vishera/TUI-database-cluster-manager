package cluster

import "context"

type ClusterManager interface {
	// Switchover - плановая передача роли лидера
	Switchover(ctx context.Context, masterNode, candidateNode string) error

	// Failover - принудительная смена лидера при аварии
	Failover(ctx context.Context, candidateNode string) error

	// Reinitialize - переинициализация реплики
	Reinitialize(ctx context.Context, node string) error

	// TogglePause - переключение режима обслуживания (Pause / Resume)
	TogglePause(ctx context.Context) (bool, error)
}

package cluster

import (
	"context"
	"time"
)

type NodeRole string

const (
	RolePrimary NodeRole = "primary"
	RoleReplica NodeRole = "replica"
	RoleUnknown NodeRole = "unknown"
)

type NodeStatus struct {
	Name           string
	Host           string
	Port           int
	Role           NodeRole
	State          string // e.g. "online", "promoted", "stopped", "failed"
	Timeline       int
	ReplicationLag uint64
	IsOnline       bool
}

type ClusterStatus struct {
	EngineName  string // "Patroni", "Corosync+Pacemaker", "Repmgr"
	Scope       string
	Nodes       []NodeStatus
	Maintenance bool
	LastUpdated time.Time
}

// Provider определяет абстрактный интерфейс для взаимодействия с HA-инструментами
type Provider interface {
	// Engine returns the engine identifier (e.g. "corosync-pacemaker")
	Engine() string

	// GetStatus retrieves current cluster topology and state
	GetStatus(ctx context.Context) (*ClusterStatus, error)

	// Switchover triggers graceful primary role transfer to candidate node
	Switchover(ctx context.Context, currentPrimary, candidate string) error

	// Reinitialize resets/resyncs a replica node
	Reinitialize(ctx context.Context, targetNode string) error

	// SetMaintenance Toggles maintenance / pause mode
	SetMaintenance(ctx context.Context, enabled bool) error
}

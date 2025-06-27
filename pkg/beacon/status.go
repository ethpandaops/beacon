package beacon

import (
	"sync"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
)

// Status is a beacon node status.
type Status struct {
	mu        sync.RWMutex
	health    *Health
	networkID uint64
	syncstate *v1.SyncState
}

// NewStatus creates a new status.
func NewStatus(successThreshold, failThreshold int) *Status {
	return &Status{
		health:    NewHealth(successThreshold, failThreshold),
		networkID: 0,
		syncstate: nil,
	}
}

// Healthy returns true if the beacon node is healthy.
func (s *Status) Healthy() bool {
	return s.health.Healthy()
}

// Health returns the health status.
func (s *Status) Health() *Health {
	return s.health
}

// NetworkID returns the network ID.
func (s *Status) NetworkID() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.networkID
}

// Syncing returns true if the beacon node is syncing.
func (s *Status) Syncing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.syncstate == nil {
		return false
	}

	return s.syncstate.IsSyncing
}

// SyncState returns the sync state.
func (s *Status) SyncState() *v1.SyncState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.syncstate
}

// UpdateNetworkID updates the network ID.
func (s *Status) UpdateNetworkID(networkID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.networkID = networkID
}

// UpdateSyncState updates the sync state.
func (s *Status) UpdateSyncState(state *v1.SyncState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncstate = state
}

package beacon

import v1 "github.com/attestantio/go-eth2-client/api/v1"

type Status struct {
	health    *Health
	networkID uint64
	syncstate *v1.SyncState
}

func NewStatus(successThreshold, failThreshold int) *Status {
	return &Status{
		health:    NewHealth(successThreshold, failThreshold),
		networkID: 0,
		syncstate: nil,
	}
}

func (s *Status) Healthy() bool {
	return s.health.Healthy()
}

func (s *Status) Health() *Health {
	return s.health
}

func (s *Status) NetworkID() uint64 {
	return s.networkID
}

func (s *Status) Syncing() bool {
	if s.syncstate == nil {
		return false
	}

	return s.syncstate.IsSyncing
}

func (s *Status) SyncState() *v1.SyncState {
	return s.syncstate
}

func (s *Status) UpdateNetworkID(networkID uint64) {
	s.networkID = networkID
}

func (s *Status) UpdateSyncState(state *v1.SyncState) {
	s.syncstate = state
}

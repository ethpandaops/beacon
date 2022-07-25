package beacon

import v1 "github.com/attestantio/go-eth2-client/api/v1"

type Status struct {
	healthy   Health
	finality  *v1.Finality
	networkID uint64
	syncstate *v1.SyncState
}

func NewStatus(successThreshold, failThreshold int) Status {
	return Status{
		healthy:   NewHealth(successThreshold, failThreshold),
		finality:  nil,
		networkID: 0,
		syncstate: nil,
	}
}

func (s *Status) Healthy() bool {
	return s.healthy.Healthy()
}

func (s *Status) Health() Health {
	return s.healthy
}

func (s *Status) Finality() *v1.Finality {
	return s.finality
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

func (s *Status) UpdateFinality(finality *v1.Finality) {
	s.finality = finality
}

func (s *Status) UpdateNetworkID(networkID uint64) {
	s.networkID = networkID
}

func (s *Status) UpdateSyncState(state *v1.SyncState) {
	s.syncstate = state
}

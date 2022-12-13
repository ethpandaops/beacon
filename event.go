package beacon

import (
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/samcm/beacon/api/types"
	"github.com/samcm/beacon/state"
)

type EventTopics []string

func (e EventTopics) Exists(topic string) bool {
	for _, t := range e {
		if t == topic {
			return true
		}
	}

	return false
}

const (
	// Custom events derived from our pseudo beacon node
	topicEpochChanged              = "epoch_changed"
	topicSlotChanged               = "slot_changed"
	topicEpochSlotChanged          = "epoch_slot_changed"
	topicReady                     = "ready"
	topicSyncStatus                = "sync_status"
	topicNodeVersionUpdated        = "node_version_updated"
	topicPeersUpdated              = "peers_updated"
	topicSpecUpdated               = "spec_updated"
	topicEmptySlot                 = "slot_empty"
	topicHealthCheckSucceeded      = "health_check_suceeded"
	topicHealthCheckFailed         = "health_check_failed"
	topicFinalityCheckpointUpdated = "finality_checkpoint_updated"

	// Official beacon events that are proxied
	topicAttestation          = "attestation"
	topicBlock                = "block"
	topicChainReorg           = "chain_reorg"
	topicFinalizedCheckpoint  = "finalized_checkpoint"
	topicHead                 = "head"
	topicVoluntaryExit        = "voluntary_exit"
	topicContributionAndProof = "contribution_and_proof"
	topicEvent                = "raw_event"
)

type BlockInsertedEvent struct {
	Slot phase0.Slot
}

type ReadyEvent struct {
}

type SyncStatusEvent struct {
	State *v1.SyncState
}

type NodeVersionUpdatedEvent struct {
	Version string
}

type PeersUpdatedEvent struct {
	Peers types.Peers
}

type SpecUpdatedEvent struct {
	Spec *state.Spec
}

type EmptySlotEvent struct {
	Slot phase0.Slot
}

type HealthCheckSucceededEvent struct {
	Duration time.Duration
}

type HealthCheckFailedEvent struct {
	Duration time.Duration
}

type FinalityCheckpointUpdated struct {
	Finality *v1.Finality
}

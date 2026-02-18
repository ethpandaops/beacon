package beacon

import (
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

// EventTopics is a list of topics that can be subscribed to.
type EventTopics []string

// Exists returns true if the topic exists in the list.
func (e EventTopics) Exists(topic string) bool {
	for _, t := range e {
		if t == topic {
			return true
		}
	}

	return false
}

const (
	// Custom events derived from our pseudo beacon node.
	topicEpochChanged              = "epoch_changed"
	topicSlotChanged               = "slot_changed"
	topicEpochSlotChanged          = "epoch_slot_changed"
	topicReady                     = "ready"
	topicSyncStatus                = "sync_status"
	topicNodeVersionUpdated        = "node_version_updated"
	topicPeersUpdated              = "peers_updated"
	topicSpecUpdated               = "spec_updated"
	topicEmptySlot                 = "slot_empty"
	topicHealthCheckSucceeded      = "health_check_suceeded" //nolint:misspell // existing.
	topicHealthCheckFailed         = "health_check_failed"
	topicFinalityCheckpointUpdated = "finality_checkpoint_updated"
	topicFirstTimeHealthy          = "first_time_healthy"

	// Official beacon events that are proxied.
	topicAttestation          = "attestation"
	topicSingleAttestation    = "single_attestation"
	topicBlock                = "block"
	topicBlockGossip          = "block_gossip"
	topicChainReorg           = "chain_reorg"
	topicFinalizedCheckpoint  = "finalized_checkpoint"
	topicHead                 = "head"
	topicFinalized            = "finalized"
	topicVoluntaryExit        = "voluntary_exit"
	topicContributionAndProof = "contribution_and_proof"
	topicBlobSidecar          = "blob_sidecar"
	topicDataColumnSidecar    = "data_column_sidecar"
	topicEvent                = "raw_event"
)

type ReadyEvent struct {
}

// SyncStatusEvent is emitted when the sync status is refreshed.
type SyncStatusEvent struct {
	State *v1.SyncState
}

// NodeVersionUpdatedEvent is emitted when the node version is updated.
type NodeVersionUpdatedEvent struct {
	Version string
}

// PeersUpdatedEvent is emitted when the peer list is updated.
type PeersUpdatedEvent struct {
	Peers types.Peers
}

// SpecUpdatedEvent is emitted when the spec is updated.
type SpecUpdatedEvent struct {
	Spec *state.Spec
}

// EmptySlotEvent is emitted when an empty slot is detected.
type EmptySlotEvent struct {
	Slot phase0.Slot
}

// HealthCheckSucceededEvent is emitted when a health check succeeds.
type HealthCheckSucceededEvent struct {
	Duration time.Duration
}

// HealthCheckFailedEvent is emitted when a health check fails.
type HealthCheckFailedEvent struct {
	Duration time.Duration
}

// FinalityCheckpointUpdated is emitted when the finality checkpoint is updated.
type FinalityCheckpointUpdated struct {
	Finality *v1.Finality
}

// FirstTimeHealthyEvent is emitted when the node is first considered healthy.
type FirstTimeHealthyEvent struct {
}

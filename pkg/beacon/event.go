package beacon

import (
	"errors"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/prysmaticlabs/go-bitfield"
)

// EventTopics is a list of topics that can be subscribed to
type EventTopics []string

// Exists returns true if the topic exists in the list
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
	topicFirstTimeHealthy          = "first_time_healthy"

	// Official beacon events that are proxied
	topicAttestation          = "attestation"
	topicBlock                = "block"
	topicChainReorg           = "chain_reorg"
	topicFinalizedCheckpoint  = "finalized_checkpoint"
	topicHead                 = "head"
	topicVoluntaryExit        = "voluntary_exit"
	topicContributionAndProof = "contribution_and_proof"
	topicBlobSidecar          = "blob_sidecar"
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

type VersionedAttestation struct {
	Electra *electra.Attestation
	Phase0  *phase0.Attestation
	Version spec.DataVersion
}

func (v *VersionedAttestation) IsElectra() bool {
	return v.Version == spec.DataVersionElectra
}

func (v *VersionedAttestation) IsPhase0() bool {
	return v.Version == spec.DataVersionPhase0
}

func (v *VersionedAttestation) IsValid() bool {
	return v.IsElectra() || v.IsPhase0()
}

func (v *VersionedAttestation) GetVersion() spec.DataVersion {
	return v.Version
}

func (v *VersionedAttestation) AggregationBits() (bitfield.Bitlist, error) {
	if v.IsElectra() {
		return v.Electra.AggregationBits, nil
	}

	if v.IsPhase0() {
		return v.Phase0.AggregationBits, nil
	}

	return nil, errors.New("invalid attestation")
}

func (v *VersionedAttestation) Slot() (phase0.Slot, error) {
	if v.IsElectra() {
		return v.Electra.Data.Slot, nil
	}

	if v.IsPhase0() {
		return v.Phase0.Data.Slot, nil
	}

	return 0, errors.New("invalid attestation")
}

func (v *VersionedAttestation) Target() (*phase0.Checkpoint, error) {
	if v.IsElectra() {
		return v.Electra.Data.Target, nil
	}

	if v.IsPhase0() {
		return v.Phase0.Data.Target, nil
	}

	return nil, errors.New("invalid attestation")
}

func (v *VersionedAttestation) Source() (*phase0.Checkpoint, error) {
	if v.IsElectra() {
		return v.Electra.Data.Source, nil
	}

	if v.IsPhase0() {
		return v.Phase0.Data.Source, nil
	}

	return nil, errors.New("invalid attestation")
}

func (v *VersionedAttestation) Signature() (phase0.BLSSignature, error) {
	if v.IsElectra() {
		return v.Electra.Signature, nil
	}

	if v.IsPhase0() {
		return v.Phase0.Signature, nil
	}

	return phase0.BLSSignature{}, errors.New("invalid attestation")
}

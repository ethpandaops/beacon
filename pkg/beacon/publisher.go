package beacon

import (
	"context"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

// Official beacon events that are proxied.
func (n *node) publishBlock(ctx context.Context, event *v1.BlockEvent) {
	n.broker.Emit(topicBlock, event)
}

func (n *node) publishBlockGossip(ctx context.Context, event *v1.BlockGossipEvent) {
	n.broker.Emit(topicBlockGossip, event)
}

func (n *node) publishAttestation(ctx context.Context, event *spec.VersionedAttestation) {
	n.broker.Emit(topicAttestation, event)
}

func (n *node) publishChainReOrg(ctx context.Context, event *v1.ChainReorgEvent) {
	n.broker.Emit(topicChainReorg, event)
}

func (n *node) publishFinalizedCheckpoint(ctx context.Context, event *v1.FinalizedCheckpointEvent) {
	n.broker.Emit(topicFinalizedCheckpoint, event)
}

func (n *node) publishHead(ctx context.Context, event *v1.HeadEvent) {
	n.broker.Emit(topicHead, event)
}

func (n *node) publishVoluntaryExit(ctx context.Context, event *phase0.SignedVoluntaryExit) {
	n.broker.Emit(topicVoluntaryExit, event)
}

func (n *node) publishContributionAndProof(ctx context.Context, event *altair.SignedContributionAndProof) {
	n.broker.Emit(topicContributionAndProof, event)
}

func (n *node) publishBlobSidecar(ctx context.Context, event *v1.BlobSidecarEvent) {
	n.broker.Emit(topicBlobSidecar, event)
}

func (n *node) publishDataColumnSidecar(ctx context.Context, event *v1.DataColumnSidecarEvent) {
	n.broker.Emit(topicDataColumnSidecar, event)
}

func (n *node) publishEvent(ctx context.Context, event *v1.Event) {
	n.broker.Emit(topicEvent, event)
}

// Custom Events derived from our pseudo beacon node.
func (n *node) publishReady(ctx context.Context) {
	n.broker.Emit(topicReady, nil)
}

func (n *node) publishSyncStatus(ctx context.Context, st *v1.SyncState) {
	n.broker.Emit(topicSyncStatus, &SyncStatusEvent{
		State: st,
	})
}

func (n *node) publishNodeVersionUpdated(ctx context.Context, version string) {
	n.broker.Emit(topicNodeVersionUpdated, &NodeVersionUpdatedEvent{
		Version: version,
	})
}

func (n *node) publishPeersUpdated(ctx context.Context, peers types.Peers) {
	n.broker.Emit(topicPeersUpdated, &PeersUpdatedEvent{
		Peers: peers,
	})
}

func (n *node) publishSpecUpdated(ctx context.Context, spec *state.Spec) {
	n.broker.Emit(topicSpecUpdated, &SpecUpdatedEvent{
		Spec: spec,
	})
}

func (n *node) publishEmptySlot(ctx context.Context, slot phase0.Slot) {
	n.broker.Emit(topicEmptySlot, &EmptySlotEvent{
		Slot: slot,
	})
}

func (n *node) publishHealthCheckSucceeded(ctx context.Context, duration time.Duration) {
	n.broker.Emit(topicHealthCheckSucceeded, &HealthCheckSucceededEvent{
		Duration: duration,
	})
}

func (n *node) publishHealthCheckFailed(ctx context.Context, duration time.Duration) {
	n.broker.Emit(topicHealthCheckFailed, &HealthCheckFailedEvent{
		Duration: duration,
	})
}

func (n *node) publishFinalityCheckpointUpdated(ctx context.Context, finality *v1.Finality) {
	n.broker.Emit(topicFinalityCheckpointUpdated, &FinalityCheckpointUpdated{
		Finality: finality,
	})
}

func (n *node) publishFirstTimeHealthy(ctx context.Context) {
	n.broker.Emit(topicFirstTimeHealthy, &FirstTimeHealthyEvent{})
}

func (n *node) publishSingleAttestation(ctx context.Context, event *electra.SingleAttestation) {
	n.broker.Emit(topicSingleAttestation, event)
}

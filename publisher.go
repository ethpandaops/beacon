package beacon

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/samcm/beacon/api/types"
	"github.com/samcm/beacon/state"
)

// Official beacon events that are proxied
func (n *node) publishBlock(ctx context.Context, event *v1.BlockEvent) {
	n.broker.Emit(topicBlock, event)
}

func (n *node) publishAttestation(ctx context.Context, event *phase0.Attestation) {
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

func (n *node) publishVoluntaryExit(ctx context.Context, event *phase0.VoluntaryExit) {
	n.broker.Emit(topicVoluntaryExit, event)
}

func (n *node) publishEvent(ctx context.Context, event *v1.Event) {
	n.broker.Emit(topicEvent, event)
}

// Custom Events derived from our pseudo beacon node
func (n *node) publishReady(ctx context.Context) {
	n.broker.Emit(topicReady, nil)
}

func (n *node) publishEpochChanged(ctx context.Context, epoch phase0.Epoch) {
	n.broker.Emit(topicEpochChanged, &EpochChangedEvent{
		Epoch: epoch,
	})
}

func (n *node) publishSlotChanged(ctx context.Context, slot phase0.Slot) {
	n.broker.Emit(topicSlotChanged, &SlotChangedEvent{
		Slot: slot,
	})
}

func (n *node) publishEpochSlotChanged(ctx context.Context, epoch phase0.Epoch, slot phase0.Slot) {
	n.broker.Emit(topicEpochSlotChanged, &EpochSlotChangedEvent{
		Epoch: epoch,
		Slot:  slot,
	})
}

func (n *node) publishBlockInserted(ctx context.Context, slot phase0.Slot) {
	n.broker.Emit(topicBlockInserted, &BlockInsertedEvent{
		Slot: slot,
	})
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

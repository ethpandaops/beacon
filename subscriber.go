package beacon

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (n *node) handleSubscriberError(err error, topic string) {
	if err != nil {
		n.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

// Official Beacon events
func (n *node) OnBlock(ctx context.Context, handler func(ctx context.Context, event *v1.BlockEvent) error) {
	n.broker.On(topicBlock, func(event *v1.BlockEvent) {
		n.handleSubscriberError(handler(ctx, event), topicBlock)
	})
}

func (n *node) OnAttestation(ctx context.Context, handler func(ctx context.Context, event *phase0.Attestation) error) {
	n.broker.On(topicAttestation, func(event *phase0.Attestation) {
		n.handleSubscriberError(handler(ctx, event), topicAttestation)
	})
}

func (n *node) OnChainReOrg(ctx context.Context, handler func(ctx context.Context, event *v1.ChainReorgEvent) error) {
	n.broker.On(topicChainReorg, func(event *v1.ChainReorgEvent) {
		n.handleSubscriberError(handler(ctx, event), topicChainReorg)
	})
}

func (n *node) OnFinalizedCheckpoint(ctx context.Context, handler func(ctx context.Context, event *v1.FinalizedCheckpointEvent) error) {
	n.broker.On(topicFinalizedCheckpoint, func(event *v1.FinalizedCheckpointEvent) {
		n.handleSubscriberError(handler(ctx, event), topicFinalizedCheckpoint)
	})
}

func (n *node) OnHead(ctx context.Context, handler func(ctx context.Context, event *v1.HeadEvent) error) {
	n.broker.On(topicHead, func(event *v1.HeadEvent) {
		n.handleSubscriberError(handler(ctx, event), topicHead)
	})
}

func (n *node) OnVoluntaryExit(ctx context.Context, handler func(ctx context.Context, event *phase0.VoluntaryExit) error) {
	n.broker.On(topicVoluntaryExit, func(event *phase0.VoluntaryExit) {
		n.handleSubscriberError(handler(ctx, event), topicVoluntaryExit)
	})
}

func (n *node) OnEvent(ctx context.Context, handler func(ctx context.Context, event *v1.Event) error) {
	n.broker.On(topicEvent, func(event *v1.Event) {
		n.handleSubscriberError(handler(ctx, event), topicEvent)
	})
}

// Custom Events
func (n *node) OnEpochChanged(ctx context.Context, handler func(ctx context.Context, event *EpochChangedEvent) error) {
	n.broker.On(topicEpochChanged, func(event *EpochChangedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicEpochChanged)
	})
}

func (n *node) OnSlotChanged(ctx context.Context, handler func(ctx context.Context, event *SlotChangedEvent) error) {
	n.broker.On(topicSlotChanged, func(event *SlotChangedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicSlotChanged)
	})
}

func (n *node) OnEpochSlotChanged(ctx context.Context, handler func(ctx context.Context, event *EpochSlotChangedEvent) error) {
	n.broker.On(topicEpochSlotChanged, func(event *EpochSlotChangedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicEpochSlotChanged)
	})
}

func (n *node) OnBlockInserted(ctx context.Context, handler func(ctx context.Context, event *BlockInsertedEvent) error) {
	n.broker.On(topicBlockInserted, func(event *BlockInsertedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicBlockInserted)
	})
}

func (n *node) OnReady(ctx context.Context, handler func(ctx context.Context, event *ReadyEvent) error) {
	n.broker.On(topicReady, func(event *ReadyEvent) {
		n.handleSubscriberError(handler(ctx, event), topicReady)
	})
}

func (n *node) OnSyncStatus(ctx context.Context, handler func(ctx context.Context, event *SyncStatusEvent) error) {
	n.broker.On(topicSyncStatus, func(event *SyncStatusEvent) {
		n.handleSubscriberError(handler(ctx, event), topicSyncStatus)
	})
}

func (n *node) OnNodeVersionUpdated(ctx context.Context, handler func(ctx context.Context, event *NodeVersionUpdatedEvent) error) {
	n.broker.On(topicNodeVersionUpdated, func(event *NodeVersionUpdatedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicNodeVersionUpdated)
	})
}

func (n *node) OnPeersUpdated(ctx context.Context, handler func(ctx context.Context, event *PeersUpdatedEvent) error) {
	n.broker.On(topicPeersUpdated, func(event *PeersUpdatedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicPeersUpdated)
	})
}

func (n *node) OnSpecUpdated(ctx context.Context, handler func(ctx context.Context, event *SpecUpdatedEvent) error) {
	n.broker.On(topicSpecUpdated, func(event *SpecUpdatedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicSpecUpdated)
	})
}

func (n *node) OnEmptySlot(ctx context.Context, handler func(ctx context.Context, event *EmptySlotEvent) error) {
	n.broker.On(topicEmptySlot, func(event *EmptySlotEvent) {
		n.handleSubscriberError(handler(ctx, event), topicEmptySlot)
	})
}

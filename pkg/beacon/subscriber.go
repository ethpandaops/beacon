package beacon

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (n *node) handleSubscriberError(err error, topic string) {
	if err != nil {
		n.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

// Official Beacon events.
func (n *node) OnBlock(ctx context.Context, handler func(ctx context.Context, event *v1.BlockEvent) error) {
	n.broker.On(topicBlock, func(event *v1.BlockEvent) {
		n.handleSubscriberError(handler(ctx, event), topicBlock)
	})
}

func (n *node) OnAttestation(ctx context.Context, handler func(ctx context.Context, event *spec.VersionedAttestation) error) {
	n.broker.On(topicAttestation, func(event *spec.VersionedAttestation) {
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

func (n *node) OnVoluntaryExit(ctx context.Context, handler func(ctx context.Context, event *phase0.SignedVoluntaryExit) error) {
	n.broker.On(topicVoluntaryExit, func(event *phase0.SignedVoluntaryExit) {
		n.handleSubscriberError(handler(ctx, event), topicVoluntaryExit)
	})
}

func (n *node) OnContributionAndProof(ctx context.Context, handler func(ctx context.Context, event *altair.SignedContributionAndProof) error) {
	n.broker.On(topicContributionAndProof, func(event *altair.SignedContributionAndProof) {
		n.handleSubscriberError(handler(ctx, event), topicContributionAndProof)
	})
}

func (n *node) OnBlobSidecar(ctx context.Context, handler func(ctx context.Context, event *v1.BlobSidecarEvent) error) {
	n.broker.On(topicBlobSidecar, func(event *v1.BlobSidecarEvent) {
		n.handleSubscriberError(handler(ctx, event), topicBlobSidecar)
	})
}

func (n *node) OnSingleAttestation(ctx context.Context, handler func(ctx context.Context, event *electra.SingleAttestation) error) {
	n.broker.On(topicSingleAttestation, func(event *electra.SingleAttestation) {
		n.handleSubscriberError(handler(ctx, event), topicSingleAttestation)
	})
}

func (n *node) OnEvent(ctx context.Context, handler func(ctx context.Context, event *v1.Event) error) {
	n.broker.On(topicEvent, func(event *v1.Event) {
		n.handleSubscriberError(handler(ctx, event), topicEvent)
	})
}

// Custom Events.
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

func (n *node) OnHealthCheckFailed(ctx context.Context, handler func(ctx context.Context, event *HealthCheckFailedEvent) error) {
	n.broker.On(topicHealthCheckFailed, func(event *HealthCheckFailedEvent) {
		n.handleSubscriberError(handler(ctx, event), topicHealthCheckFailed)
	})
}

func (n *node) OnHealthCheckSucceeded(ctx context.Context, handler func(ctx context.Context, event *HealthCheckSucceededEvent) error) {
	n.broker.On(topicHealthCheckSucceeded, func(event *HealthCheckSucceededEvent) {
		n.handleSubscriberError(handler(ctx, event), topicHealthCheckSucceeded)
	})
}

func (n *node) OnFinalityCheckpointUpdated(ctx context.Context, handler func(ctx context.Context, event *FinalityCheckpointUpdated) error) {
	n.broker.On(topicFinalityCheckpointUpdated, func(event *FinalityCheckpointUpdated) {
		n.handleSubscriberError(handler(ctx, event), topicFinalityCheckpointUpdated)
	})
}

func (n *node) OnFirstTimeHealthy(ctx context.Context, handler func(ctx context.Context, event *FirstTimeHealthyEvent) error) {
	n.broker.On(topicFirstTimeHealthy, func(event *FirstTimeHealthyEvent) {
		n.handleSubscriberError(handler(ctx, event), topicFirstTimeHealthy)
	})
}

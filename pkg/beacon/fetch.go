package beacon

import (
	"context"
	"errors"

	eth2client "github.com/attestantio/go-eth2-client"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

func (n *node) FetchSyncStatus(ctx context.Context) (*v1.SyncState, error) {
	provider, isProvider := n.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	status, err := provider.NodeSyncing(ctx)
	if err != nil {
		return nil, err
	}

	n.stat.UpdateSyncState(status)

	n.publishSyncStatus(ctx, status)

	return status, nil
}

func (n *node) FetchPeers(ctx context.Context) (*types.Peers, error) {
	peers, err := n.api.NodePeers(ctx)
	if err != nil {
		return nil, err
	}

	n.peers = peers

	n.publishPeersUpdated(ctx, peers)

	return &peers, nil
}

func (n *node) FetchNodeVersion(ctx context.Context) (string, error) {
	provider, isProvider := n.client.(eth2client.NodeVersionProvider)
	if !isProvider {
		return "", errors.New("client does not implement eth2client.NodeVersionProvider")
	}

	version, err := provider.NodeVersion(ctx)
	if err != nil {
		return "", err
	}

	n.nodeVersion = version

	n.publishNodeVersionUpdated(ctx, version)

	return version, nil
}

func (n *node) FetchBlock(ctx context.Context, stateID string) (*spec.VersionedSignedBeaconBlock, error) {
	return n.getBlock(ctx, stateID)
}

func (n *node) FetchBeaconState(ctx context.Context, stateID string) (*spec.VersionedBeaconState, error) {
	provider, isProvider := n.client.(eth2client.BeaconStateProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.NodeVersionProvider")
	}

	beaconState, err := provider.BeaconState(ctx, stateID)
	if err != nil {
		return nil, err
	}

	return beaconState, nil
}

func (n *node) FetchRawBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	return n.api.RawDebugBeaconState(ctx, stateID, contentType)
}

func (n *node) FetchFinality(ctx context.Context, stateID string) (*v1.Finality, error) {
	provider, isProvider := n.client.(eth2client.FinalityProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.FinalityProvider")
	}

	finality, err := provider.Finality(ctx, stateID)
	if err != nil {
		return nil, err
	}

	if stateID == "head" {
		changed := false
		if n.finality == nil ||
			finality.Finalized.Root != n.finality.Finalized.Root ||
			finality.Finalized.Epoch != n.finality.Finalized.Epoch ||
			finality.Justified.Root != n.finality.Justified.Root ||
			finality.Justified.Epoch != n.finality.Justified.Epoch ||
			finality.PreviousJustified.Epoch != n.finality.PreviousJustified.Epoch ||
			finality.PreviousJustified.Root != n.finality.PreviousJustified.Root {
			changed = true
		}

		n.finality = finality

		if changed {
			n.publishFinalityCheckpointUpdated(ctx, finality)
		}
	}

	return finality, nil
}

func (n *node) FetchSpec(ctx context.Context) (*state.Spec, error) {
	provider, isProvider := n.client.(eth2client.SpecProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.SpecProvider")
	}

	data, err := provider.Spec(ctx)
	if err != nil {
		return nil, err
	}

	sp := state.NewSpec(data)

	n.spec = &sp

	n.publishSpecUpdated(ctx, &sp)

	return &sp, nil
}

func (n *node) FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
	n.log.WithField("epoch", epoch).Debug("Fetching proposer duties")

	provider, isProvider := n.client.(eth2client.ProposerDutiesProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.ProposerDutiesProvider")
	}

	duties, err := provider.ProposerDuties(ctx, epoch, nil)
	if err != nil {
		return nil, err
	}

	return duties, nil
}

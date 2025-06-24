package beacon

import (
	"context"
	"errors"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

func (n *node) FetchSyncStatus(ctx context.Context) (*v1.SyncState, error) {
	provider, isProvider := n.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	status, err := provider.NodeSyncing(ctx, &api.NodeSyncingOpts{})
	if err != nil {
		return nil, err
	}

	n.stat.UpdateSyncState(status.Data)

	n.publishSyncStatus(ctx, status.Data)

	return status.Data, nil
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

	rsp, err := provider.NodeVersion(ctx, &api.NodeVersionOpts{})
	if err != nil {
		return "", err
	}

	n.nodeVersionMu.Lock()
	n.nodeVersion = rsp.Data
	n.nodeVersionMu.Unlock()

	n.publishNodeVersionUpdated(ctx, rsp.Data)

	return rsp.Data, nil
}

func (n *node) FetchBlock(ctx context.Context, stateID string) (*spec.VersionedSignedBeaconBlock, error) {
	return n.getBlock(ctx, stateID)
}

func (n *node) FetchRawBlock(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	return n.api.RawBlock(ctx, stateID, contentType)
}

func (n *node) FetchBlockRoot(ctx context.Context, stateID string) (*phase0.Root, error) {
	return n.getBlockRoot(ctx, stateID)
}

func (n *node) FetchBeaconState(ctx context.Context, stateID string) (*spec.VersionedBeaconState, error) {
	provider, isProvider := n.client.(eth2client.BeaconStateProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.NodeVersionProvider")
	}

	rsp, err := provider.BeaconState(ctx, &api.BeaconStateOpts{
		State: stateID,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchRawBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	return n.api.RawDebugBeaconState(ctx, stateID, contentType)
}

func (n *node) FetchFinality(ctx context.Context, stateID string) (*v1.Finality, error) {
	provider, isProvider := n.client.(eth2client.FinalityProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.FinalityProvider")
	}

	rsp, err := provider.Finality(ctx, &api.FinalityOpts{
		State: stateID,
	})
	if err != nil {
		return nil, err
	}

	finality := rsp.Data

	//nolint:goconst // existing.
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

func (n *node) FetchRawSpec(ctx context.Context) (map[string]any, error) {
	provider, isProvider := n.client.(eth2client.SpecProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.SpecProvider")
	}

	rsp, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchSpec(ctx context.Context) (*state.Spec, error) {
	provider, isProvider := n.client.(eth2client.SpecProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.SpecProvider")
	}

	rsp, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		return nil, err
	}

	sp := state.NewSpec(rsp.Data)

	n.specMu.Lock()
	n.spec = &sp
	n.specMu.Unlock()

	n.publishSpecUpdated(ctx, &sp)

	return &sp, nil
}

func (n *node) FetchBeaconBlockBlobs(ctx context.Context, blockID string) ([]*deneb.BlobSidecar, error) {
	provider, isProvider := n.client.(eth2client.BlobSidecarsProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.BlobSidecarsProvider")
	}

	rsp, err := provider.BlobSidecars(ctx, &api.BlobSidecarsOpts{
		Block: blockID,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
	n.log.WithField("epoch", epoch).Debug("Fetching proposer duties")

	provider, isProvider := n.client.(eth2client.ProposerDutiesProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.ProposerDutiesProvider")
	}

	rsp, err := provider.ProposerDuties(ctx, &api.ProposerDutiesOpts{
		Epoch: epoch,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchForkChoice(ctx context.Context) (*v1.ForkChoice, error) {
	provider, isProvider := n.client.(eth2client.ForkChoiceProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.ForkChoiceProvider")
	}

	rsp, err := provider.ForkChoice(ctx, &api.ForkChoiceOpts{})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchDepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error) {
	return n.api.DepositSnapshot(ctx)
}

func (n *node) FetchNodeIdentity(ctx context.Context) (*types.Identity, error) {
	return n.api.NodeIdentity(ctx)
}

func (n *node) FetchBeaconStateRoot(ctx context.Context, state string) (phase0.Root, error) {
	provider, isProvider := n.client.(eth2client.BeaconStateRootProvider)
	if !isProvider {
		return phase0.Root{}, errors.New("client does not implement eth2client.StateRootProvider")
	}

	rsp, err := provider.BeaconStateRoot(ctx, &api.BeaconStateRootOpts{
		State: state,
	})
	if err != nil {
		return phase0.Root{}, err
	}

	return *rsp.Data, nil
}

func (n *node) FetchValidators(ctx context.Context, state string, indices []phase0.ValidatorIndex, pubKeys []phase0.BLSPubKey) (map[phase0.ValidatorIndex]*v1.Validator, error) {
	provider, isProvider := n.client.(eth2client.ValidatorsProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.ValidatorsProvider")
	}

	rsp, err := provider.Validators(ctx, &api.ValidatorsOpts{
		State:   state,
		Indices: indices,
		PubKeys: pubKeys,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchBeaconCommittees(ctx context.Context, state string, epoch *phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	provider, isProvider := n.client.(eth2client.BeaconCommitteesProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.BeaconCommitteesProvider")
	}

	opts := &api.BeaconCommitteesOpts{
		State: state,
	}

	if epoch != nil {
		opts.Epoch = epoch
	}

	rsp, err := provider.BeaconCommittees(ctx, opts)
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchAttestationData(ctx context.Context, slot phase0.Slot, committeeIndex phase0.CommitteeIndex) (*phase0.AttestationData, error) {
	provider, isProvider := n.client.(eth2client.AttestationDataProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.AttestationDataProvider")
	}

	rsp, err := provider.AttestationData(ctx, &api.AttestationDataOpts{
		Slot:           slot,
		CommitteeIndex: committeeIndex,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (n *node) FetchBeaconBlockHeader(ctx context.Context, opts *api.BeaconBlockHeaderOpts) (*v1.BeaconBlockHeader, error) {
	provider, isProvider := n.client.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.BeaconBlockHeadersProvider")
	}

	rsp, err := provider.BeaconBlockHeader(ctx, opts)
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

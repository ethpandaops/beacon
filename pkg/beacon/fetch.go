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
	logCtx := n.log.WithField("method", "FetchSyncStatus")
	provider, isProvider := n.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		logCtx.Error("client does not implement eth2client.NodeSyncingProvider")

		return nil, errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	logCtx.Debug("Fetching sync status")

	status, err := provider.NodeSyncing(ctx, &api.NodeSyncingOpts{})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch sync status")

		return nil, err
	}

	n.stat.UpdateSyncState(status.Data)

	n.publishSyncStatus(ctx, status.Data)

	logCtx.WithField("status", status.Data).Debug("Successfully fetched sync status")

	return status.Data, nil
}

func (n *node) FetchPeers(ctx context.Context) (*types.Peers, error) {
	logCtx := n.log.WithField("method", "FetchPeers")

	logCtx.Debug("Fetching peers")

	peers, err := n.api.NodePeers(ctx)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch peers")

		return nil, err
	}

	logCtx.WithField("peers", len(peers)).Debug("Successfully fetched peers")

	n.peers = peers

	n.publishPeersUpdated(ctx, peers)

	return &peers, nil
}

func (n *node) FetchNodeVersion(ctx context.Context) (string, error) {
	logCtx := n.log.WithField("method", "FetchNodeVersion")

	logCtx.Debug("Fetching node version")

	provider, isProvider := n.client.(eth2client.NodeVersionProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.NodeVersionProvider")
		logCtx.WithError(err).Error("client does not implement eth2client.NodeVersionProvider")

		return "", err
	}

	rsp, err := provider.NodeVersion(ctx, &api.NodeVersionOpts{})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch node version")

		return "", err
	}

	n.nodeVersion = rsp.Data

	n.publishNodeVersionUpdated(ctx, rsp.Data)

	logCtx.WithField("version", rsp.Data).Debug("Successfully fetched node version")

	return rsp.Data, nil
}

func (n *node) FetchBlock(ctx context.Context, stateID string) (*spec.VersionedSignedBeaconBlock, error) {
	logCtx := n.log.WithField("method", "FetchBlock").WithField("state_id", stateID)

	logCtx.Debug("Fetching block")

	block, err := n.getBlock(ctx, stateID)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch block")

		return nil, err
	}

	logCtx.Debug("Successfully fetched block")

	return block, nil
}

func (n *node) FetchRawBlock(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	logCtx := n.log.WithField("method", "FetchRawBlock").WithField("state_id", stateID)

	logCtx.Debug("Fetching raw block")

	block, err := n.api.RawBlock(ctx, stateID, contentType)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch raw block")

		return nil, err
	}

	logCtx.Debug("Successfully fetched raw block")

	return block, nil
}

func (n *node) FetchBlockRoot(ctx context.Context, stateID string) (*phase0.Root, error) {
	logCtx := n.log.WithField("method", "FetchBlockRoot").WithField("state_id", stateID)

	logCtx.Debug("Fetching block root")

	root, err := n.getBlockRoot(ctx, stateID)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch block root")

		return nil, err
	}

	logCtx.Debug("Successfully fetched block root")

	return root, nil
}

func (n *node) FetchBeaconState(ctx context.Context, stateID string) (*spec.VersionedBeaconState, error) {
	logCtx := n.log.WithField("method", "FetchBeaconState").WithField("state_id", stateID)

	provider, isProvider := n.client.(eth2client.BeaconStateProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.NodeVersionProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	logCtx.Debug("Fetching beacon state")

	rsp, err := provider.BeaconState(ctx, &api.BeaconStateOpts{
		State: stateID,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch beacon state")

		return nil, err
	}

	logCtx.Debug("Successfully fetched beacon state")

	return rsp.Data, nil
}

func (n *node) FetchRawBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	logCtx := n.log.WithField("method", "FetchRawBeaconState").WithField("state_id", stateID)

	logCtx.Debug("Fetching raw beacon state")

	block, err := n.api.RawDebugBeaconState(ctx, stateID, contentType)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch raw beacon state")

		return nil, err
	}

	logCtx.Debug("Successfully fetched raw beacon state")

	return block, nil
}

func (n *node) FetchFinality(ctx context.Context, stateID string) (*v1.Finality, error) {
	logCtx := n.log.WithField("method", "FetchFinality").WithField("state_id", stateID)

	provider, isProvider := n.client.(eth2client.FinalityProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.FinalityProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	logCtx.Debug("Fetching finality")

	rsp, err := provider.Finality(ctx, &api.FinalityOpts{
		State: stateID,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch finality")

		return nil, err
	}

	finality := rsp.Data

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

	logCtx.Debug("Successfully fetched finality")

	return finality, nil
}

func (n *node) FetchRawSpec(ctx context.Context) (map[string]any, error) {
	logCtx := n.log.WithField("method", "FetchRawSpec")

	logCtx.Debug("Fetching raw spec")

	provider, isProvider := n.client.(eth2client.SpecProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.SpecProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch raw spec")

		return nil, err
	}

	logCtx.Debug("Successfully fetched raw spec")

	return rsp.Data, nil
}

func (n *node) FetchSpec(ctx context.Context) (*state.Spec, error) {
	logCtx := n.log.WithField("method", "FetchSpec")

	logCtx.Debug("Fetching spec")

	provider, isProvider := n.client.(eth2client.SpecProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.SpecProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch spec")

		return nil, err
	}

	sp := state.NewSpec(rsp.Data)

	n.spec = &sp

	n.publishSpecUpdated(ctx, &sp)

	logCtx.Debug("Successfully fetched spec")

	return &sp, nil
}

func (n *node) FetchBeaconBlockBlobs(ctx context.Context, blockID string) ([]*deneb.BlobSidecar, error) {
	logCtx := n.log.WithField("method", "FetchBeaconBlockBlobs").WithField("block_id", blockID)

	logCtx.Debug("Fetching beacon blobs")

	provider, isProvider := n.client.(eth2client.BlobSidecarsProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.BlobSidecarsProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.BlobSidecars(ctx, &api.BlobSidecarsOpts{
		Block: blockID,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch beacon blobs")

		return nil, err
	}

	logCtx.WithField("blob_count", len(rsp.Data)).Debug("Successfully fetched beacon blobs")

	return rsp.Data, nil
}

func (n *node) FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
	logCtx := n.log.WithField("method", "FetchProposerDuties").WithField("epoch", epoch)

	logCtx.Debug("Fetching proposer duties")

	provider, isProvider := n.client.(eth2client.ProposerDutiesProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.ProposerDutiesProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.ProposerDuties(ctx, &api.ProposerDutiesOpts{
		Epoch: epoch,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch proposer duties")

		return nil, err
	}

	logCtx.Debug("Successfully fetched proposer duties")

	return rsp.Data, nil
}

func (n *node) FetchForkChoice(ctx context.Context) (*v1.ForkChoice, error) {
	logCtx := n.log.WithField("method", "FetchForkChoice")

	logCtx.Debug("Fetching fork choice")

	provider, isProvider := n.client.(eth2client.ForkChoiceProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.ForkChoiceProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.ForkChoice(ctx, &api.ForkChoiceOpts{})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch fork choice")

		return nil, err
	}

	logCtx.Debug("Successfully fetched fork choice")

	return rsp.Data, nil
}

func (n *node) FetchDepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error) {
	logCtx := n.log.WithField("method", "FetchDepositSnapshot")

	logCtx.Debug("Fetching deposit snapshot")

	snapshot, err := n.api.DepositSnapshot(ctx)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch deposit snapshot")

		return nil, err
	}

	logCtx.Debug("Successfully fetched deposit snapshot")

	return snapshot, nil
}

func (n *node) FetchNodeIdentity(ctx context.Context) (*types.Identity, error) {
	logCtx := n.log.WithField("method", "FetchNodeIdentity")

	logCtx.Debug("Fetching node identity")

	identity, err := n.api.NodeIdentity(ctx)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch node identity")

		return nil, err
	}

	logCtx.WithField("identity", identity).Debug("Successfully fetched node identity")

	return identity, nil
}

func (n *node) FetchBeaconStateRoot(ctx context.Context, state string) (phase0.Root, error) {
	logCtx := n.log.WithField("method", "FetchBeaconStateRoot").WithField("state", state)

	logCtx.Debug("Fetching beacon state root")

	provider, isProvider := n.client.(eth2client.BeaconStateRootProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.StateRootProvider")

		logCtx.Error(err.Error())

		return phase0.Root{}, err
	}

	rsp, err := provider.BeaconStateRoot(ctx, &api.BeaconStateRootOpts{
		State: state,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch beacon state root")

		return phase0.Root{}, err
	}

	logCtx.Debug("Successfully fetched beacon state root")

	return *rsp.Data, nil
}

func (n *node) FetchBeaconCommittees(ctx context.Context, state string, epoch *phase0.Epoch) ([]*v1.BeaconCommittee, error) {
	logCtx := n.log.WithField("method", "FetchBeaconCommittees").WithField("state", state)

	logCtx.Debug("Fetching beacon committees")

	provider, isProvider := n.client.(eth2client.BeaconCommitteesProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.BeaconCommitteesProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	opts := &api.BeaconCommitteesOpts{
		State: state,
	}

	if epoch != nil {
		opts.Epoch = epoch
	}

	rsp, err := provider.BeaconCommittees(ctx, opts)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch beacon committees")

		return nil, err
	}

	logCtx.WithField("committee_count", len(rsp.Data)).Debug("Successfully fetched beacon committees")

	return rsp.Data, nil
}

func (n *node) FetchAttestationData(ctx context.Context, slot phase0.Slot, committeeIndex phase0.CommitteeIndex) (*phase0.AttestationData, error) {
	logCtx := n.log.WithField("method", "FetchAttestationData").WithField("slot", slot).WithField("committee_index", committeeIndex)

	logCtx.Debug("Fetching attestation data")

	provider, isProvider := n.client.(eth2client.AttestationDataProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.AttestationDataProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.AttestationData(ctx, &api.AttestationDataOpts{
		Slot:           slot,
		CommitteeIndex: committeeIndex,
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch attestation data")

		return nil, err
	}

	logCtx.Debug("Successfully fetched attestation data")

	return rsp.Data, nil
}

func (n *node) FetchBeaconBlockHeader(ctx context.Context, opts *api.BeaconBlockHeaderOpts) (*v1.BeaconBlockHeader, error) {
	logCtx := n.log.WithField("method", "FetchBeaconBlockHeader")

	logCtx.Debug("Fetching beacon block header")

	provider, isProvider := n.client.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		err := errors.New("client does not implement eth2client.BeaconBlockHeadersProvider")

		logCtx.Error(err.Error())

		return nil, err
	}

	rsp, err := provider.BeaconBlockHeader(ctx, opts)
	if err != nil {
		logCtx.WithError(err).Error("failed to fetch beacon block header")

		return nil, err
	}

	logCtx.Debug("Successfully fetched beacon block header")

	return rsp.Data, nil
}

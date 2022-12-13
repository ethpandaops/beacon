package beacon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/ethwallclock"
	"github.com/go-co-op/gocron"
	"github.com/samcm/beacon/api"
	"github.com/samcm/beacon/api/types"
	"github.com/samcm/beacon/state"
	"github.com/sirupsen/logrus"
)

type Node interface {
	// Lifecycle
	// Start starts the node.
	Start(ctx context.Context) error
	// StartAsync starts the node asynchronously.
	StartAsync(ctx context.Context)

	// Getters
	// Options returns the options for the node.
	Options() *Options

	// Wallclock returns the EthWallclock instance
	Wallclock() *ethwallclock.EthereumBeaconChain
	// Eth getters
	// GetSpec returns the spec for the node.
	GetSpec(ctx context.Context) (*state.Spec, error)
	// GetSyncState returns the sync state for the node.
	GetSyncState(ctx context.Context) (*v1.SyncState, error)
	// GetGenesis returns the genesis for the node.
	GetGenesis(ctx context.Context) (*v1.Genesis, error)
	// GetNodeVersion returns the node version.
	GetNodeVersion(ctx context.Context) (string, error)
	// GetStatus returns the status of the ndoe.
	GetStatus(ctx context.Context) *Status
	// GetFinality returns the finality checkpoint for the node.
	GetFinality(ctx context.Context) (*v1.Finality, error)

	// FetchBlock returns the block for the given state id.
	FetchBlock(ctx context.Context, stateID string) (*spec.VersionedSignedBeaconBlock, error)
	// FetchBeaconState returns the beacon state for the given state id.
	FetchBeaconState(ctx context.Context, stateID string) (*spec.VersionedBeaconState, error)
	// FetchRawBeaconState returns the raw, unparsed beacon state for the given state id.
	FetchRawBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error)

	// Subscriptions
	// - Proxied Beacon events
	// OnEvent is called when a beacon event is received.
	OnEvent(ctx context.Context, handler func(ctx context.Context, ev *v1.Event) error)
	// OnBlock is called when a block is received.
	OnBlock(ctx context.Context, handler func(ctx context.Context, ev *v1.BlockEvent) error)
	// OnAttestation is called when an attestation is received.
	OnAttestation(ctx context.Context, handler func(ctx context.Context, ev *phase0.Attestation) error)
	// OnFinalizedCheckpoint is called when a finalized checkpoint is received.
	OnFinalizedCheckpoint(ctx context.Context, handler func(ctx context.Context, ev *v1.FinalizedCheckpointEvent) error)
	// OnHead is called when the head is received.
	OnHead(ctx context.Context, handler func(ctx context.Context, ev *v1.HeadEvent) error)
	// OnChainReOrg is called when a chain reorg is received.
	OnChainReOrg(ctx context.Context, handler func(ctx context.Context, ev *v1.ChainReorgEvent) error)
	// OnVoluntaryExit is called when a voluntary exit is received.
	OnVoluntaryExit(ctx context.Context, handler func(ctx context.Context, ev *phase0.VoluntaryExit) error)
	// OnContributionAndProof is called when a contribution and proof is received.
	OnContributionAndProof(ctx context.Context, handler func(ctx context.Context, ev *altair.SignedContributionAndProof) error)

	// - Custom events
	// OnReady is called when the node is ready.
	OnReady(ctx context.Context, handler func(ctx context.Context, event *ReadyEvent) error)
	// OnSyncStatus is called when the sync status changes.
	OnSyncStatus(ctx context.Context, handler func(ctx context.Context, event *SyncStatusEvent) error)
	// OnNodeVersionUpdated is called when the node version is updated.
	OnNodeVersionUpdated(ctx context.Context, handler func(ctx context.Context, event *NodeVersionUpdatedEvent) error)
	// OnPeersUpdated is called when the peers are updated.
	OnPeersUpdated(ctx context.Context, handler func(ctx context.Context, event *PeersUpdatedEvent) error)
	// OnSpecUpdated is called when the spec is updated.
	OnSpecUpdated(ctx context.Context, handler func(ctx context.Context, event *SpecUpdatedEvent) error)
	// OnEmptySlot is called when an empty slot is detected.
	OnEmptySlot(ctx context.Context, handler func(ctx context.Context, event *EmptySlotEvent) error)
	// OnHealthCheckFailed is called when a health check fails.
	OnHealthCheckFailed(ctx context.Context, handler func(ctx context.Context, event *HealthCheckFailedEvent) error)
	// OnHealthCheckSucceeded is called when a health check succeeds.
	OnHealthCheckSucceeded(ctx context.Context, handler func(ctx context.Context, event *HealthCheckSucceededEvent) error)
	// OnFinalityCheckpointUpdated is called when a the head finality checkpoint is updated.
	OnFinalityCheckpointUpdated(ctx context.Context, handler func(ctx context.Context, event *FinalityCheckpointUpdated) error)
}

// Node represents an Ethereum beacon node. It computes values based on the spec.
type node struct {
	// Helpers
	log logrus.FieldLogger

	// Configuration
	// Config should roughly be driven by end users.
	config *Config
	// Options should be driven by code.
	options *Options

	// Clients
	api    api.ConsensusClient
	client eth2client.Service
	broker *emission.Emitter

	// Internal data stores
	genesis       *v1.Genesis
	lastEventTime time.Time
	nodeVersion   string
	peers         types.Peers
	finality      *v1.Finality
	spec          *state.Spec
	wallclock     *ethwallclock.EthereumBeaconChain

	status *Status

	metrics *Metrics
}

func NewNode(log logrus.FieldLogger, config *Config, namespace string, options Options) Node {
	n := &node{
		log: log.WithField("module", "consensus/beacon"),

		config:  config,
		options: &options,

		broker: emission.NewEmitter(),

		status: NewStatus(options.HealthCheck.SuccessfulResponses, options.HealthCheck.FailedResponses),
	}

	if options.PrometheusMetrics {
		n.metrics = NewMetrics(n.log, namespace+"_beacon", config.Name, n)
	}

	return n
}

func (n *node) Start(ctx context.Context) error {
	if n.options.PrometheusMetrics {
		if err := n.metrics.Start(ctx); err != nil {
			return err
		}
	}

	if err := n.ensureClients(ctx); err != nil {
		return err
	}

	if err := n.bootstrap(ctx); err != nil {
		return err
	}

	if err := n.fetchSyncStatus(ctx); err != nil {
		return err
	}

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every(n.options.HealthCheck.Interval.String()).Do(func() {
		n.runHealthcheck(ctx)
	}); err != nil {
		return err
	}

	if _, err := s.Every("15s").Do(func() {
		if err := n.fetchSyncStatus(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch sync status")
		}
	}); err != nil {
		return err
	}

	if _, err := s.Every("15m").Do(func() {
		if err := n.fetchNodeVersion(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch node version")
		}
	}); err != nil {
		return err
	}

	if _, err := s.Every("60s").Do(func() {
		if err := n.fetchPeers(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch peers")
		}
	}); err != nil {
		return err
	}

	if _, err := s.Every("30s").Do(func() {
		if err := n.fetchFinality(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch finality")
		}
	}); err != nil {
		return err
	}

	s.StartAsync()

	return nil
}

func (n *node) StartAsync(ctx context.Context) {
	go func() {
		if err := n.Start(ctx); err != nil {
			n.log.WithError(err).Error("Failed to start beacon node")
		}
	}()
}

func (n *node) Options() *Options {
	return n.options
}

func (n *node) Wallclock() *ethwallclock.EthereumBeaconChain {
	return n.wallclock
}

func (n *node) GetSpec(ctx context.Context) (*state.Spec, error) {
	if n.spec == nil {
		return nil, errors.New("spec is not available")
	}

	return n.spec, nil
}

func (n *node) GetSyncState(ctx context.Context) (*v1.SyncState, error) {
	state := n.status.SyncState()

	if state == nil {
		return nil, errors.New("sync state not available")
	}

	return state, nil
}

func (n *node) GetGenesis(ctx context.Context) (*v1.Genesis, error) {
	return n.genesis, nil
}

func (n *node) GetNodeVersion(ctx context.Context) (string, error) {
	return n.nodeVersion, nil
}

func (n *node) GetStatus(ctx context.Context) *Status {
	return n.status
}

func (n *node) GetFinality(ctx context.Context) (*v1.Finality, error) {
	if n.finality == nil {
		return nil, errors.New("finality not available")
	}

	return n.finality, nil
}

func (n *node) bootstrap(ctx context.Context) error {
	if err := n.initializeState(ctx); err != nil {
		return err
	}

	if err := n.subscribeDownstream(ctx); err != nil {
		return err
	}

	n.publishReady(ctx)

	//nolint:errcheck // we dont care if this errors out since it runs indefinitely in a goroutine
	go n.ensureBeaconSubscription(ctx)

	return nil
}

func (n *node) fetchSyncStatus(ctx context.Context) error {
	provider, isProvider := n.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	status, err := provider.NodeSyncing(ctx)
	if err != nil {
		return err
	}

	n.status.UpdateSyncState(status)

	n.publishSyncStatus(ctx, status)

	return nil
}

func (n *node) fetchPeers(ctx context.Context) error {
	peers, err := n.api.NodePeers(ctx)
	if err != nil {
		return err
	}

	n.peers = peers

	n.publishPeersUpdated(ctx, peers)

	return nil
}

func (n *node) subscribeDownstream(ctx context.Context) error {
	n.wallclock.OnEpochChanged(func(epoch ethwallclock.Epoch) {
		time.Sleep(time.Second * 3)

		if err := n.fetchFinality(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch finality")
		}
	})

	n.wallclock.OnSlotChanged(func(slot ethwallclock.Slot) {
		if !n.options.DetectEmptySlots {
			return
		}

		if n.status.Syncing() {
			return
		}

		_, err := n.FetchBlock(ctx, fmt.Sprintf("%v", slot.Number()-1))
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				n.publishEmptySlot(ctx, phase0.Slot(slot.Number()))
			}

			return
		}
	})

	return nil
}

func (n *node) fetchNodeVersion(ctx context.Context) error {
	provider, isProvider := n.client.(eth2client.NodeVersionProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.NodeVersionProvider")
	}

	version, err := provider.NodeVersion(ctx)
	if err != nil {
		return err
	}

	n.nodeVersion = version

	n.publishNodeVersionUpdated(ctx, version)

	return nil
}

func (n *node) fetchHealthy(ctx context.Context) error {
	provider, isProvider := n.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	_, err := provider.NodeSyncing(ctx)
	if err != nil {
		return err
	}

	return nil
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

func (n *node) runHealthcheck(ctx context.Context) {
	start := time.Now()

	err := n.fetchHealthy(ctx)
	if err != nil {
		n.status.Health().RecordFail(err)

		n.publishHealthCheckFailed(ctx, time.Since(start))

		return
	}

	n.status.Health().RecordSuccess()

	n.publishHealthCheckSucceeded(ctx, time.Since(start))
}

func (n *node) fetchFinality(ctx context.Context) error {
	provider, isProvider := n.client.(eth2client.FinalityProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.FinalityProvider")
	}

	finality, err := provider.Finality(ctx, "head")
	if err != nil {
		return err
	}

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

	return nil
}

func (n *node) initializeState(ctx context.Context) error {
	n.log.Info("Initializing beacon state")

	spec, err := n.fetchSpec(ctx)
	if err != nil {
		return err
	}

	genesis, err := n.fetchGenesis(ctx)
	if err != nil {
		return err
	}

	n.wallclock = ethwallclock.NewEthereumBeaconChain(genesis.GenesisTime, spec.SecondsPerSlot.AsDuration(), uint64(spec.SlotsPerEpoch))

	n.log.Info("Beacon state initialized! Ready to serve requests...")

	return nil
}

func (n *node) fetchSpec(ctx context.Context) (*state.Spec, error) {
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

func (n *node) getProserDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error) {
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

func (n *node) getBlock(ctx context.Context, blockID string) (*spec.VersionedSignedBeaconBlock, error) {
	provider, isProvider := n.client.(eth2client.SignedBeaconBlockProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.SignedBeaconBlockProvider")
	}

	signedBeaconBlock, err := provider.SignedBeaconBlock(ctx, blockID)
	if err != nil {
		return nil, err
	}

	return signedBeaconBlock, nil
}

func (n *node) Status() *Status {
	return n.status
}

func (n *node) Healthy() bool {
	return n.status.Healthy()
}

func (n *node) NetworkID() uint64 {
	return n.status.NetworkID()
}

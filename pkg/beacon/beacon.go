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
	"github.com/ethpandaops/beacon/pkg/beacon/api"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/ethwallclock"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type Node interface {
	// Lifecycle
	// Start starts the node.
	Start(ctx context.Context) error
	// StartAsync starts the node asynchronously.
	StartAsync(ctx context.Context)
	// Stop stops the node.
	Stop(ctx context.Context) error

	// Getters
	// Options returns the options for the node.
	Options() *Options

	// Wallclock returns the EthWallclock instance
	Wallclock() *ethwallclock.EthereumBeaconChain

	// Eth getters. These are all cached.
	// Spec returns the spec for the node.
	Spec() (*state.Spec, error)
	// SyncState returns the sync state for the node.
	SyncState() (*v1.SyncState, error)
	// Genesis returns the genesis for the node.
	Genesis() (*v1.Genesis, error)
	// NodeVersion returns the node version.
	NodeVersion() (string, error)
	// Status returns the status of the ndoe.
	Status() *Status
	// Finality returns the finality checkpoint for the node.
	Finality() (*v1.Finality, error)
	// Healthy returns true if the node is healthy.
	Healthy() bool

	// Fetchers - these are not cached and will always fetch from the node.
	// FetchBlock fetches the block for the given state id.
	FetchBlock(ctx context.Context, stateID string) (*spec.VersionedSignedBeaconBlock, error)
	// FetchBeaconState fetches the beacon state for the given state id.
	FetchBeaconState(ctx context.Context, stateID string) (*spec.VersionedBeaconState, error)
	// FetchRawBeaconState fetches the raw, unparsed beacon state for the given state id.
	FetchRawBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error)
	// FetchFinality fetches the finality checkpoint for the state id.
	FetchFinality(ctx context.Context, stateID string) (*v1.Finality, error)
	// FetchGenesis fetches the genesis configuration.
	FetchGenesis(ctx context.Context) (*v1.Genesis, error)
	// FetchPeers fetches the peers from the beacon node.
	FetchPeers(ctx context.Context) (*types.Peers, error)
	// FetchSyncStatus fetches the sync status from the beacon node.
	FetchSyncStatus(ctx context.Context) (*v1.SyncState, error)
	// FetchNodeVersion fetches the node version from the beacon node.
	FetchNodeVersion(ctx context.Context) (string, error)
	// FetchSpec fetches the spec from the beacon node.
	FetchSpec(ctx context.Context) (*state.Spec, error)
	// FetchProposerDuties fetches the proposer duties from the beacon node.
	FetchProposerDuties(ctx context.Context, epoch phase0.Epoch) ([]*v1.ProposerDuty, error)
	// FetchForkChoice fetches the fork choice context.
	FetchForkChoice(ctx context.Context) (*v1.ForkChoice, error)
	// FetchDepositSnapshot fetches the deposit snapshot.
	FetchDepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error)
	// FetchBeaconCommittees fetches the committees for the given epoch at the given state.
	FetchBeaconCommittees(ctx context.Context, state string, epoch phase0.Epoch) ([]*v1.BeaconCommittee, error)

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
	OnVoluntaryExit(ctx context.Context, handler func(ctx context.Context, ev *phase0.SignedVoluntaryExit) error)
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
	log    logrus.FieldLogger
	ctx    context.Context
	cancel context.CancelFunc

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

	stat *Status

	metrics *Metrics

	crons *gocron.Scheduler
}

// NewNode creates a new beacon node.
func NewNode(log logrus.FieldLogger, config *Config, namespace string, options Options) Node {
	n := &node{
		log: log.WithField("module", "consensus/beacon"),

		config:  config,
		options: &options,

		broker: emission.NewEmitter(),

		stat: NewStatus(options.HealthCheck.SuccessfulResponses, options.HealthCheck.FailedResponses),
	}

	if options.PrometheusMetrics {
		if namespace == "" {
			namespace = "eth"
		}

		n.metrics = NewMetrics(n.log, namespace, config.Name, n)
	}

	return n
}

func (n *node) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	n.ctx = ctx
	n.cancel = cancel

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

	if _, err := n.FetchSyncStatus(ctx); err != nil {
		return err
	}

	if _, err := n.FetchFinality(ctx, "head"); err != nil {
		n.log.WithError(err).Error("Failed to fetch initial head finality")
	}

	s := gocron.NewScheduler(time.Local)

	if _, err := s.Every(n.options.HealthCheck.Interval.String()).Do(func() {
		n.runHealthcheck(ctx)
	}); err != nil {
		return err
	}

	if _, err := s.Every("15s").Do(func() {
		if _, err := n.FetchSyncStatus(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch sync status")
		}
	}); err != nil {
		return err
	}

	if _, err := s.Every("15m").Do(func() {
		if _, err := n.FetchNodeVersion(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch node version")
		}
	}); err != nil {
		return err
	}

	if _, err := s.Every("60s").Do(func() {
		if _, err := n.FetchPeers(ctx); err != nil {
			n.log.WithError(err).Debug("Failed to fetch peers")
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

func (n *node) Stop(ctx context.Context) error {
	if n.options.PrometheusMetrics {
		if err := n.metrics.Stop(); err != nil {
			return err
		}
	}

	if n.crons != nil {
		n.crons.Stop()
	}

	if n.cancel != nil {
		n.cancel()
	}

	return nil
}

func (n *node) Options() *Options {
	return n.options
}

func (n *node) Wallclock() *ethwallclock.EthereumBeaconChain {
	return n.wallclock
}

func (n *node) Spec() (*state.Spec, error) {
	if n.spec == nil {
		return nil, errors.New("spec is not available")
	}

	return n.spec, nil
}

func (n *node) SyncState() (*v1.SyncState, error) {
	state := n.stat.SyncState()

	if state == nil {
		return nil, errors.New("sync state not available")
	}

	return state, nil
}

func (n *node) Genesis() (*v1.Genesis, error) {
	return n.genesis, nil
}

func (n *node) NodeVersion() (string, error) {
	return n.nodeVersion, nil
}

func (n *node) Status() *Status {
	return n.stat
}

func (n *node) Finality() (*v1.Finality, error) {
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

func (n *node) subscribeDownstream(ctx context.Context) error {
	n.wallclock.OnEpochChanged(func(epoch ethwallclock.Epoch) {
		time.Sleep(time.Second * 3)

		if _, err := n.FetchFinality(ctx, "head"); err != nil {
			n.log.WithError(err).Debug("Failed to fetch finality")
		}
	})

	n.wallclock.OnSlotChanged(func(slot ethwallclock.Slot) {
		if !n.options.DetectEmptySlots {
			return
		}

		if n.stat.Syncing() {
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

	n.OnFinalizedCheckpoint(ctx, func(ctx context.Context, ev *v1.FinalizedCheckpointEvent) error {
		time.Sleep(3 * time.Second) // Sleep to give time for the beacon node to update its state.

		if _, err := n.FetchFinality(ctx, "head"); err != nil {
			n.log.WithError(err).Debug("Failed to fetch finality for head state")
		}

		return nil
	})

	return nil
}

func (n *node) fetchIsHealthy(ctx context.Context) error {
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

func (n *node) runHealthcheck(ctx context.Context) {
	start := time.Now()

	err := n.fetchIsHealthy(ctx)
	if err != nil {
		n.stat.Health().RecordFail(err)

		n.publishHealthCheckFailed(ctx, time.Since(start))

		return
	}

	n.stat.Health().RecordSuccess()

	n.publishHealthCheckSucceeded(ctx, time.Since(start))
}

func (n *node) initializeState(ctx context.Context) error {
	n.log.Info("Initializing beacon state")

	spec, err := n.FetchSpec(ctx)
	if err != nil {
		return err
	}

	genesis, err := n.FetchGenesis(ctx)
	if err != nil {
		return err
	}

	n.wallclock = ethwallclock.NewEthereumBeaconChain(genesis.GenesisTime, spec.SecondsPerSlot.AsDuration(), uint64(spec.SlotsPerEpoch))

	n.log.Info("Beacon state initialized! Ready to serve requests...")

	return nil
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

func (n *node) Healthy() bool {
	return n.stat.Healthy()
}

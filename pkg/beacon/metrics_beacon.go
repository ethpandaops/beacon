package beacon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Beacon reports Beacon information about the beacon chain.
type BeaconMetrics struct {
	log                 logrus.FieldLogger
	beaconNode          Node
	Slot                prometheus.GaugeVec
	Transactions        prometheus.GaugeVec
	Slashings           prometheus.GaugeVec
	Attestations        prometheus.GaugeVec
	Deposits            prometheus.GaugeVec
	VoluntaryExits      prometheus.GaugeVec
	FinalityCheckpoints prometheus.GaugeVec
	ReOrgs              prometheus.Counter
	ReOrgDepth          prometheus.Counter
	EmptySlots          prometheus.Counter
	ProposerDelay       prometheus.Histogram
	Withdrawals         prometheus.GaugeVec
	WithdrawalsAmount   prometheus.GaugeVec
	WithdrawalsIndexMax prometheus.GaugeVec
	WithdrawalsIndexMin prometheus.GaugeVec
	BlobKZGCommitments  prometheus.GaugeVec

	currentVersionHead      string
	currentVersionFinalized string

	gloasHeadMu  sync.Mutex
	gloasHead    *gloasHeadBlock
	slotHookOnce sync.Once

	crons *gocron.Scheduler
}

// gloasHeadBlock is the current Gloas head block. Its payload is revealed after
// the block, so payload metrics are read once its slot has passed.
type gloasHeadBlock struct {
	root     phase0.Root
	slot     phase0.Slot
	version  string
	attempts int
	inFlight bool
	resolved bool
}

// gloasHeadPayloadAttempts is how many slot boundaries a head block's envelope is
// looked for before its payload is treated as withheld. Reveals can land late in
// the slot and reach the node after the next boundary.
const gloasHeadPayloadAttempts = 3

const (
	metricsJobNameBeacon = "beacon"
)

// NewBeaconMetrics creates a new BeaconMetrics instance.
func NewBeaconMetrics(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *BeaconMetrics {
	constLabels["module"] = metricsJobNameBeacon
	namespace += "_beacon"

	b := &BeaconMetrics{
		beaconNode: beac,
		log:        log,
		crons:      gocron.NewScheduler(time.Local),
		Slot: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "slot",
				Help:        "The slot number in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		Transactions: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "transactions",
				Help:        "The amount of transactions in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		Slashings: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "slashings",
				Help:        "The amount of slashings in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
				"type",
			},
		),
		Attestations: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "attestations",
				Help:        "The amount of attestations in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		Deposits: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "deposits",
				Help:        "The amount of deposits in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		VoluntaryExits: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "voluntary_exits",
				Help:        "The amount of voluntary exits in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		FinalityCheckpoints: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "finality_checkpoint_epochs",
				Help:        "That epochs of the finality checkpoints.",
				ConstLabels: constLabels,
			},
			[]string{
				"state_id",
				"checkpoint",
			},
		),
		ReOrgs: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "reorg_count",
				Help:        "The count of reorgs.",
				ConstLabels: constLabels,
			},
		),
		ReOrgDepth: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "reorg_depth",
				Help:        "The number of reorgs.",
				ConstLabels: constLabels,
			},
		),
		ProposerDelay: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace:   namespace,
				Name:        "proposer_delay",
				Help:        "The delay of the proposer.",
				ConstLabels: constLabels,
				Buckets:     prometheus.LinearBuckets(0, 1000, 13),
			},
		),
		EmptySlots: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "empty_slots_count",
				Help:        "The number of slots that have expired without a block proposed.",
				ConstLabels: constLabels,
			},
		),
		Withdrawals: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "withdrawals",
				Help:        "The amount of withdrawals in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		WithdrawalsAmount: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "withdrawals_amount_gwei",
				Help:        "The sum amount of all the withdrawals in the block (in gwei).",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		WithdrawalsIndexMax: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "withdrawals_index_max",
				Help:        "The maximum index of the withdrawals in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		WithdrawalsIndexMin: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "withdrawals_index_min",
				Help:        "The minimum index of the withdrawals in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
		BlobKZGCommitments: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "blob_kzg_commitments",
				Help:        "The amount of blob kzg commitments in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				metricLabelBlockID,
				metricLabelVersion,
			},
		),
	}

	prometheus.MustRegister(b.Attestations)
	prometheus.MustRegister(b.Deposits)
	prometheus.MustRegister(b.Slashings)
	prometheus.MustRegister(b.Transactions)
	prometheus.MustRegister(b.VoluntaryExits)
	prometheus.MustRegister(b.Slot)
	prometheus.MustRegister(b.FinalityCheckpoints)
	prometheus.MustRegister(b.ReOrgs)
	prometheus.MustRegister(b.ReOrgDepth)
	prometheus.MustRegister(b.ProposerDelay)
	prometheus.MustRegister(b.EmptySlots)
	prometheus.MustRegister(b.Withdrawals)
	prometheus.MustRegister(b.WithdrawalsAmount)
	prometheus.MustRegister(b.WithdrawalsIndexMax)
	prometheus.MustRegister(b.WithdrawalsIndexMin)
	prometheus.MustRegister(b.BlobKZGCommitments)

	return b
}

// Name returns the name of the job.
func (b *BeaconMetrics) Name() string {
	return metricsJobNameBeacon
}

// Start starts the job.
func (b *BeaconMetrics) Start(ctx context.Context) error {
	b.beaconNode.OnReady(ctx, func(ctx context.Context, event *ReadyEvent) error {
		b.slotHookOnce.Do(func() {
			b.beaconNode.Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
				b.recordGloasHeadPayload(ctx, phase0.Slot(slot.Number()))
			})
		})

		time.Sleep(3 * time.Second)

		return b.updateFinality(ctx)
	})

	if err := b.setupSubscriptions(ctx); err != nil {
		return err
	}

	b.crons.StartAsync()

	return nil
}

// Stop stops the job.
func (b *BeaconMetrics) Stop() error {
	b.crons.Stop()

	return nil
}

func (b *BeaconMetrics) setupSubscriptions(ctx context.Context) error {
	b.beaconNode.OnBlock(ctx, b.handleBlock)

	b.beaconNode.OnBlock(ctx, func(ctx context.Context, event *v1.BlockEvent) error {
		syncState, err := b.beaconNode.SyncState()
		if err != nil {
			return err
		}

		if syncState == nil || syncState.IsSyncing {
			return nil
		}

		block, err := b.beaconNode.FetchBlock(ctx, fmt.Sprintf("%#x", event.Block))
		if err != nil {
			return err
		}

		if err := b.handleSingleBlock(ctx, "head", block); err != nil {
			return err
		}

		return nil
	})

	b.beaconNode.OnChainReOrg(ctx, b.handleChainReorg)

	b.beaconNode.OnEmptySlot(ctx, b.handleEmptySlot)

	b.beaconNode.OnFinalityCheckpointUpdated(ctx, func(ctx context.Context, ev *FinalityCheckpointUpdated) error {
		return b.updateFinality(ctx)
	})

	return nil
}

func (b *BeaconMetrics) handleEmptySlot(ctx context.Context, event *EmptySlotEvent) error {
	syncState, err := b.beaconNode.SyncState()
	if err != nil {
		return err
	}

	if syncState == nil || syncState.IsSyncing {
		return nil
	}

	b.log.WithField("slot", event.Slot).Debug("Empty slot detected")

	b.EmptySlots.Inc()

	return nil
}

func (b *BeaconMetrics) handleBlock(ctx context.Context, event *v1.BlockEvent) error {
	syncState, err := b.beaconNode.SyncState()
	if err != nil {
		return nil //nolint:nilerr // existing.
	}

	if syncState == nil || syncState.IsSyncing {
		return nil
	}

	slot := b.beaconNode.Wallclock().Slots().FromNumber(uint64(event.Slot))

	currSlot, _, err := b.beaconNode.Wallclock().Now()
	if err != nil {
		return err
	}

	// We don't care about blocks that are more than 2 slots in the past.
	if currSlot.Number()-slot.Number() > 2 {
		return nil
	}

	delay := time.Since(slot.TimeWindow().Start())

	b.ProposerDelay.Observe(float64(delay.Milliseconds()))

	return nil
}

func (b *BeaconMetrics) handleChainReorg(ctx context.Context, event *v1.ChainReorgEvent) error {
	b.ReOrgs.Inc()
	b.ReOrgDepth.Add(float64(event.Depth))

	return nil
}

func (b *BeaconMetrics) GetSignedBeaconBlock(ctx context.Context, blockID string) error {
	block, err := b.beaconNode.FetchBlock(ctx, blockID)
	if err != nil {
		return err
	}

	if err := b.handleSingleBlock(ctx, blockID, block); err != nil {
		return err
	}

	return nil
}

// updateFinality updates the finality metrics.
func (b *BeaconMetrics) updateFinality(ctx context.Context) error {
	if err := b.GetSignedBeaconBlock(ctx, "finalized"); err != nil {
		b.log.WithError(err).Error("Failed to get signed beacon block at finalized")
	}

	if err := b.GetSignedBeaconBlock(ctx, "head"); err != nil {
		b.log.WithError(err).Error("Failed to get signed beacon block at head")
	}

	finality, err := b.beaconNode.Finality()
	if err != nil {
		return err
	}

	b.FinalityCheckpoints.
		WithLabelValues("head", "previous_justified").
		Set(float64(finality.PreviousJustified.Epoch))

	b.FinalityCheckpoints.
		WithLabelValues("head", "justified").
		Set(float64(finality.Justified.Epoch))

	b.FinalityCheckpoints.
		WithLabelValues("head", "finalized").
		Set(float64(finality.Finalized.Epoch))

	return nil
}

func (b *BeaconMetrics) handleSingleBlock(ctx context.Context, blockID string, block *spec.VersionedSignedBeaconBlock) error {
	if block == nil {
		return errors.New("block is nil")
	}

	if blockID == topicHead && b.currentVersionHead != block.Version.String() ||
		blockID == topicFinalized && b.currentVersionFinalized != block.Version.String() {
		blockLabels := prometheus.Labels{metricLabelBlockID: blockID}

		for _, vec := range []*prometheus.GaugeVec{
			&b.Transactions,
			&b.Slashings,
			&b.Attestations,
			&b.Deposits,
			&b.VoluntaryExits,
			&b.Slot,
			&b.Withdrawals,
			&b.WithdrawalsAmount,
			&b.WithdrawalsIndexMax,
			&b.WithdrawalsIndexMin,
			&b.BlobKZGCommitments,
		} {
			vec.DeletePartialMatch(blockLabels)
		}

		if blockID == topicFinalized {
			b.currentVersionFinalized = block.Version.String()
		}

		if blockID == topicHead {
			b.currentVersionHead = block.Version.String()
		}
	}

	b.recordNewBeaconBlock(blockID, block)

	if block.Version >= spec.DataVersionGloas {
		b.handleGloasBlockPayload(ctx, blockID, block)
	} else if blockID == topicHead {
		b.gloasHeadMu.Lock()
		b.gloasHead = nil
		b.gloasHeadMu.Unlock()
	}

	return nil
}

// handleGloasBlockPayload records payload metrics for a Gloas block. The block no
// longer carries its execution payload: transactions and withdrawals live in an
// envelope revealed after the block, which a builder can also withhold.
func (b *BeaconMetrics) handleGloasBlockPayload(ctx context.Context, blockID string, block *spec.VersionedSignedBeaconBlock) {
	version := block.Version.String()

	root, err := block.Root()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get root from block")

		return
	}

	if blockID != topicHead {
		b.applyPayload(blockID, version, b.fetchPayload(ctx, blockID, root))

		return
	}

	slot, err := block.Slot()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get slot from block")

		return
	}

	b.gloasHeadMu.Lock()
	defer b.gloasHeadMu.Unlock()

	if b.gloasHead != nil && b.gloasHead.root == root {
		return
	}

	b.gloasHead = &gloasHeadBlock{
		root:    root,
		slot:    slot,
		version: version,
	}
}

// recordGloasHeadPayload records the head block's payload metrics once the chain
// has moved past the head block's slot, after the payload reveal deadline. Until
// then the metrics keep describing the previous head's payload. A missing envelope
// is looked for again on later slots while the block stays head, and only treated
// as withheld once the attempts run out.
func (b *BeaconMetrics) recordGloasHeadPayload(ctx context.Context, currentSlot phase0.Slot) {
	b.gloasHeadMu.Lock()

	head := b.gloasHead
	if head == nil || head.resolved || head.inFlight || head.slot >= currentSlot {
		b.gloasHeadMu.Unlock()

		return
	}

	head.inFlight = true
	head.attempts++
	root, version := head.root, head.version

	b.gloasHeadMu.Unlock()

	payload := b.fetchPayload(ctx, topicHead, root)

	b.gloasHeadMu.Lock()
	defer b.gloasHeadMu.Unlock()

	head.inFlight = false

	if b.gloasHead != head {
		return
	}

	if payload == nil && head.attempts < gloasHeadPayloadAttempts {
		return
	}

	head.resolved = true

	b.applyPayload(topicHead, version, payload)
}

// applyPayload records the payload metrics, or removes them when the block has no
// payload, so a withheld payload never reports the previous block's values.
func (b *BeaconMetrics) applyPayload(blockID, version string, payload *gloas.ExecutionPayload) {
	if payload == nil {
		b.deletePayloadMetrics(blockID, version)

		return
	}

	b.recordPayload(blockID, version, payload.Transactions, payload.Withdrawals)
}

// fetchPayload returns the execution payload of the block with the given root, or
// nil when the node has none for it.
func (b *BeaconMetrics) fetchPayload(ctx context.Context, blockID string, root phase0.Root) *gloas.ExecutionPayload {
	envelope, err := b.beaconNode.FetchExecutionPayloadEnvelope(ctx, fmt.Sprintf("%#x", root))
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Warn("Failed to fetch execution payload envelope")

		return nil
	}

	if envelope == nil {
		return nil
	}

	payload, err := envelope.Payload()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Warn("Failed to get payload from execution payload envelope")

		return nil
	}

	return payload
}

func (b *BeaconMetrics) deletePayloadMetrics(blockID, version string) {
	b.Transactions.DeleteLabelValues(blockID, version)
	b.Withdrawals.DeleteLabelValues(blockID, version)
	b.WithdrawalsAmount.DeleteLabelValues(blockID, version)
	b.WithdrawalsIndexMax.DeleteLabelValues(blockID, version)
	b.WithdrawalsIndexMin.DeleteLabelValues(blockID, version)
}

func (b *BeaconMetrics) recordPayload(blockID, version string, transactions []bellatrix.Transaction, withdrawals []*capella.Withdrawal) {
	b.Transactions.WithLabelValues(blockID, version).Set(float64(len(transactions)))
	b.recordWithdrawals(blockID, version, withdrawals)
}

func (b *BeaconMetrics) recordWithdrawals(blockID, version string, withdrawals []*capella.Withdrawal) {
	var gwei uint64

	var indexMax uint64

	indexMin := uint64(math.MaxUint64)

	for _, withdrawal := range withdrawals {
		gwei += uint64(withdrawal.Amount)

		index := uint64(withdrawal.Index)
		if index > indexMax {
			indexMax = index
		}

		if index < indexMin {
			indexMin = index
		}
	}

	b.WithdrawalsAmount.WithLabelValues(blockID, version).Set(float64(gwei))
	b.Withdrawals.WithLabelValues(blockID, version).Set(float64(len(withdrawals)))

	if indexMax > 0 {
		b.WithdrawalsIndexMax.WithLabelValues(blockID, version).Set(float64(indexMax))
	}

	if indexMin < math.MaxUint64 {
		b.WithdrawalsIndexMin.WithLabelValues(blockID, version).Set(float64(indexMin))
	}
}

func (b *BeaconMetrics) recordNewBeaconBlock(blockID string, block *spec.VersionedSignedBeaconBlock) {
	version := block.Version.String()

	slot, err := block.Slot()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get slot from block")
	} else {
		b.Slot.WithLabelValues(blockID, version).Set(float64(slot))
	}

	attesterSlashing, err := block.AttesterSlashings()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get attester slashing from block")
	} else {
		b.Slashings.WithLabelValues(blockID, version, "attester").Set(float64(len(attesterSlashing)))
	}

	proposerSlashing, err := block.ProposerSlashings()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get proposer slashing from block")
	} else {
		b.Slashings.WithLabelValues(blockID, version, "proposer").Set(float64(len(proposerSlashing)))
	}

	attestations, err := block.Attestations()
	if err != nil {
		b.log.WithError(err).WithField(metricLabelBlockID, blockID).Error("Failed to get attestations from block")
	} else {
		b.Attestations.WithLabelValues(blockID, version).Set(float64(len(attestations)))
	}

	deposits := GetDepositCountsFromBeaconBlock(block)
	b.Deposits.WithLabelValues(blockID, version).Set(float64(deposits))

	voluntaryExits := GetVoluntaryExitsFromBeaconBlock(block)
	b.VoluntaryExits.WithLabelValues(blockID, version).Set(float64(voluntaryExits))

	if block.Version < spec.DataVersionGloas {
		transactions := GetTransactionsCountFromBeaconBlock(block)
		b.Transactions.WithLabelValues(blockID, version).Set(float64(transactions))

		if withdrawals, withdrawalsErr := block.Withdrawals(); withdrawalsErr == nil {
			b.recordWithdrawals(blockID, version, withdrawals)
		}
	}

	blobs, err := block.BlobKZGCommitments()
	if err == nil {
		b.BlobKZGCommitments.WithLabelValues(blockID, version).Set(float64(len(blobs)))
	}
}

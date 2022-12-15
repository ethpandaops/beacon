package beacon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
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
	currentVersion      string
}

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
		Slot: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "slot",
				Help:        "The slot number in the block.",
				ConstLabels: constLabels,
			},
			[]string{
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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
				"block_id",
				"version",
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

	return b
}

// Name returns the name of the job.
func (b *BeaconMetrics) Name() string {
	return metricsJobNameBeacon
}

// Start starts the job.
func (b *BeaconMetrics) Start(ctx context.Context) error {
	b.beaconNode.OnReady(ctx, func(ctx context.Context, event *ReadyEvent) error {
		time.Sleep(3 * time.Second)

		b.updateFinalizedCheckpoint(ctx)

		return nil
	})

	if err := b.setupSubscriptions(ctx); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 60):
				b.tick(ctx)
			}
		}
	}()

	return nil
}

func (b *BeaconMetrics) tick(ctx context.Context) {
	b.updateFinalizedCheckpoint(ctx)
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

		if err := b.handleSingleBlock("head", block); err != nil {
			return err
		}

		return nil
	})

	b.beaconNode.OnChainReOrg(ctx, b.handleChainReorg)

	b.beaconNode.OnEmptySlot(ctx, b.handleEmptySlot)

	b.beaconNode.OnFinalizedCheckpoint(ctx, func(ctx context.Context, ev *v1.FinalizedCheckpointEvent) error {
		syncState, err := b.beaconNode.SyncState()
		if err != nil {
			return err
		}

		if syncState == nil || syncState.IsSyncing {
			return nil
		}

		// Sleep for 3 seconds to allow the beacon node to process the finalized checkpoint.
		time.Sleep(3 * time.Second)

		b.updateFinalizedCheckpoint(ctx)

		return nil
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
		return nil
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

func (b *BeaconMetrics) handleFinalizedCheckpointEvent(ctx context.Context, event *v1.FinalizedCheckpointEvent) error {
	b.updateFinalizedCheckpoint(ctx)

	return nil
}

func (b *BeaconMetrics) updateFinalizedCheckpoint(ctx context.Context) {
	if err := b.getFinality(ctx, "head"); err != nil {
		b.log.WithError(err).Error("Failed to get finality")
	}

	if err := b.GetSignedBeaconBlock(ctx, "finalized"); err != nil {
		b.log.WithError(err).Error("Failed to get signed beacon block")
	}
}

func (b *BeaconMetrics) GetSignedBeaconBlock(ctx context.Context, blockID string) error {
	block, err := b.beaconNode.FetchBlock(ctx, blockID)
	if err != nil {
		return err
	}

	if err := b.handleSingleBlock(blockID, block); err != nil {
		return err
	}

	return nil
}

// GetFinality fetches the finality checkpoints for the given state ID and records them as metrics.
func (b *BeaconMetrics) getFinality(ctx context.Context, stateID string) error {
	finality, err := b.beaconNode.FetchFinality(ctx, stateID)
	if err != nil {
		return err
	}

	b.FinalityCheckpoints.
		WithLabelValues(stateID, "previous_justified").
		Set(float64(finality.PreviousJustified.Epoch))

	b.FinalityCheckpoints.
		WithLabelValues(stateID, "justified").
		Set(float64(finality.Justified.Epoch))

	b.FinalityCheckpoints.
		WithLabelValues(stateID, "finalized").
		Set(float64(finality.Finalized.Epoch))

	return nil
}

func (b *BeaconMetrics) handleSingleBlock(blockID string, block *spec.VersionedSignedBeaconBlock) error {
	if block == nil {
		return errors.New("block is nil")
	}

	if blockID == "head" && b.currentVersion != block.Version.String() {
		b.Transactions.Reset()
		b.Slashings.Reset()
		b.Attestations.Reset()
		b.Deposits.Reset()
		b.VoluntaryExits.Reset()
		b.Slot.Reset()

		b.currentVersion = block.Version.String()
	}

	b.recordNewBeaconBlock(blockID, block)

	return nil
}

func (b *BeaconMetrics) recordNewBeaconBlock(blockID string, block *spec.VersionedSignedBeaconBlock) {
	version := block.Version.String()

	slot, err := block.Slot()
	if err != nil {
		b.log.WithError(err).WithField("block_id", blockID).Error("Failed to get slot from block")
	} else {
		b.Slot.WithLabelValues(blockID, version).Set(float64(slot))
	}

	attesterSlashing, err := block.AttesterSlashings()
	if err != nil {
		b.log.WithError(err).WithField("block_id", blockID).Error("Failed to get attester slashing from block")
	} else {
		b.Slashings.WithLabelValues(blockID, version, "attester").Set(float64(len(attesterSlashing)))
	}

	proposerSlashing, err := block.ProposerSlashings()
	if err != nil {
		b.log.WithError(err).WithField("block_id", blockID).Error("Failed to get proposer slashing from block")
	} else {
		b.Slashings.WithLabelValues(blockID, version, "proposer").Set(float64(len(proposerSlashing)))
	}

	attestations, err := block.Attestations()
	if err != nil {
		b.log.WithError(err).WithField("block_id", blockID).Error("Failed to get attestations from block")
	} else {
		b.Attestations.WithLabelValues(blockID, version).Set(float64(len(attestations)))
	}

	deposits := GetDepositCountsFromBeaconBlock(block)
	b.Deposits.WithLabelValues(blockID, version).Set(float64(deposits))

	voluntaryExits := GetVoluntaryExitsFromBeaconBlock(block)
	b.VoluntaryExits.WithLabelValues(blockID, version).Set(float64(voluntaryExits))

	transactions := GetTransactionsCountFromBeaconBlock(block)
	b.Transactions.WithLabelValues(blockID, version).Set(float64(transactions))

	if block.Version == spec.DataVersionCapella {
		gwei := int64(0)
		indexMax := int64(0)
		indexMin := int64(math.MaxInt64)

		for _, withdrawal := range block.Capella.Message.Body.ExecutionPayload.Withdrawals {
			gwei += int64(withdrawal.Amount)

			index := int64(withdrawal.Index)
			if index > indexMax {
				indexMax = index
			}

			if index < indexMin {
				indexMin = index
			}
		}

		b.WithdrawalsAmount.WithLabelValues(blockID, version).Set(float64(gwei))
		b.Withdrawals.WithLabelValues(blockID, version).Set(float64(len(block.Capella.Message.Body.ExecutionPayload.Withdrawals)))

		if indexMax > 0 {
			b.WithdrawalsIndexMax.WithLabelValues(blockID, version).Set(float64(indexMax))
		}

		if indexMin < math.MaxInt64 {
			b.WithdrawalsIndexMin.WithLabelValues(blockID, version).Set(float64(indexMin))
		}
	}
}

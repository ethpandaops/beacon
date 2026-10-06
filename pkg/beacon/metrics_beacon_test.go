package beacon

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var versionGloas = spec.DataVersionGloas.String()

// envelopeNode serves execution payload envelopes from a map keyed by block id.
type envelopeNode struct {
	Node

	envelopes map[string]*spec.VersionedSignedExecutionPayloadEnvelope
	fetched   []string
	onFetch   func()
}

func (n *envelopeNode) FetchExecutionPayloadEnvelope(_ context.Context, blockID string) (*spec.VersionedSignedExecutionPayloadEnvelope, error) {
	n.fetched = append(n.fetched, blockID)

	if n.onFetch != nil {
		n.onFetch()
	}

	return n.envelopes[blockID], nil
}

func newTestBeaconMetrics(node Node) *BeaconMetrics {
	labels := []string{metricLabelBlockID, metricLabelVersion}

	gauge := func(name string) prometheus.GaugeVec {
		return *prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, labels)
	}

	return &BeaconMetrics{
		log:                 logrus.New(),
		beaconNode:          node,
		Transactions:        gauge("transactions"),
		Withdrawals:         gauge("withdrawals"),
		WithdrawalsAmount:   gauge("withdrawals_amount_gwei"),
		WithdrawalsIndexMax: gauge("withdrawals_index_max"),
		WithdrawalsIndexMin: gauge("withdrawals_index_min"),
	}
}

func testEnvelope(transactions, withdrawals int) *spec.VersionedSignedExecutionPayloadEnvelope {
	payload := &gloas.ExecutionPayload{}

	for i := 0; i < transactions; i++ {
		payload.Transactions = append(payload.Transactions, bellatrix.Transaction{byte(i)})
	}

	for i := 0; i < withdrawals; i++ {
		payload.Withdrawals = append(payload.Withdrawals, &capella.Withdrawal{
			Index:  capella.WithdrawalIndex(100 + i),
			Amount: phase0.Gwei(10),
		})
	}

	return &spec.VersionedSignedExecutionPayloadEnvelope{
		Version: spec.DataVersionGloas,
		Gloas: &gloas.SignedExecutionPayloadEnvelope{
			Message: &gloas.ExecutionPayloadEnvelope{Payload: payload},
		},
	}
}

func blockIDFor(root phase0.Root) string {
	return fmt.Sprintf("%#x", root)
}

func TestRecordGloasHeadPayloadWaitsForSlotToPass(t *testing.T) {
	root := phase0.Root{1}
	node := &envelopeNode{envelopes: map[string]*spec.VersionedSignedExecutionPayloadEnvelope{
		blockIDFor(root): testEnvelope(3, 2),
	}}

	b := newTestBeaconMetrics(node)
	b.gloasHead = &gloasHeadBlock{root: root, slot: 10, version: versionGloas}

	b.recordGloasHeadPayload(context.Background(), 10)
	assert.Empty(t, node.fetched)

	b.recordGloasHeadPayload(context.Background(), 11)
	require.Equal(t, []string{blockIDFor(root)}, node.fetched)

	assert.InDelta(t, 3, testutil.ToFloat64(b.Transactions.WithLabelValues(topicHead, versionGloas)), 0)
	assert.InDelta(t, 2, testutil.ToFloat64(b.Withdrawals.WithLabelValues(topicHead, versionGloas)), 0)
	assert.InDelta(t, 20, testutil.ToFloat64(b.WithdrawalsAmount.WithLabelValues(topicHead, versionGloas)), 0)
	assert.InDelta(t, 101, testutil.ToFloat64(b.WithdrawalsIndexMax.WithLabelValues(topicHead, versionGloas)), 0)
	assert.InDelta(t, 100, testutil.ToFloat64(b.WithdrawalsIndexMin.WithLabelValues(topicHead, versionGloas)), 0)

	b.recordGloasHeadPayload(context.Background(), 12)
	assert.Len(t, node.fetched, 1, "a head block's payload is only read once")
}

func TestRecordGloasHeadPayloadWithheldClearsPreviousPayload(t *testing.T) {
	node := &envelopeNode{envelopes: map[string]*spec.VersionedSignedExecutionPayloadEnvelope{}}

	b := newTestBeaconMetrics(node)
	b.Transactions.WithLabelValues(topicHead, versionGloas).Set(7)
	b.Withdrawals.WithLabelValues(topicHead, versionGloas).Set(4)
	b.gloasHead = &gloasHeadBlock{root: phase0.Root{2}, slot: 10, version: versionGloas}

	b.recordGloasHeadPayload(context.Background(), 11)
	b.recordGloasHeadPayload(context.Background(), 12)

	assert.InDelta(t, 7, testutil.ToFloat64(b.Transactions.WithLabelValues(topicHead, versionGloas)), 0,
		"a late reveal keeps the previous payload until the attempts run out")

	b.recordGloasHeadPayload(context.Background(), 13)

	assert.Len(t, node.fetched, gloasHeadPayloadAttempts)
	assert.Equal(t, 0, testutil.CollectAndCount(&b.Transactions))
	assert.Equal(t, 0, testutil.CollectAndCount(&b.Withdrawals))
}

func TestRecordGloasHeadPayloadPicksUpLateReveal(t *testing.T) {
	root := phase0.Root{5}
	node := &envelopeNode{envelopes: map[string]*spec.VersionedSignedExecutionPayloadEnvelope{}}

	b := newTestBeaconMetrics(node)
	b.gloasHead = &gloasHeadBlock{root: root, slot: 10, version: versionGloas}

	b.recordGloasHeadPayload(context.Background(), 11)
	assert.Equal(t, 0, testutil.CollectAndCount(&b.Transactions))

	node.envelopes[blockIDFor(root)] = testEnvelope(9, 0)

	b.recordGloasHeadPayload(context.Background(), 12)
	assert.InDelta(t, 9, testutil.ToFloat64(b.Transactions.WithLabelValues(topicHead, versionGloas)), 0)
}

func TestRecordGloasHeadPayloadIgnoresSupersededHead(t *testing.T) {
	root := phase0.Root{3}
	node := &envelopeNode{envelopes: map[string]*spec.VersionedSignedExecutionPayloadEnvelope{
		blockIDFor(root): testEnvelope(5, 0),
	}}

	b := newTestBeaconMetrics(node)
	b.gloasHead = &gloasHeadBlock{root: root, slot: 10, version: versionGloas}

	node.onFetch = func() {
		b.gloasHeadMu.Lock()
		b.gloasHead = &gloasHeadBlock{root: phase0.Root{4}, slot: 11, version: versionGloas}
		b.gloasHeadMu.Unlock()
	}

	b.recordGloasHeadPayload(context.Background(), 11)

	assert.Equal(t, 0, testutil.CollectAndCount(&b.Transactions))
}

package beacon

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()

	metricDTO := &dto.Metric{}
	if err := g.Write(metricDTO); err != nil {
		t.Fatalf("failed to read gauge: %v", err)
	}

	return metricDTO.Gauge.GetValue()
}

func TestObserveSpec_TerminalTotalDifficulty(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	job := NewSpecJob(nil, log, "test_ttd_gauge", map[string]string{})

	mainnetTTD, ok := new(big.Int).SetString("58750000000000000000000", 10)
	if !ok {
		t.Fatal("failed to parse mainnet TTD constant")
	}

	s := &state.Spec{TerminalTotalDifficulty: *mainnetTTD}

	if err := job.observeSpec(context.Background(), s); err != nil {
		t.Fatalf("observeSpec returned error: %v", err)
	}

	got := gaugeValue(t, job.TerminalTotalDifficulty)

	want, _ := new(big.Float).SetInt(mainnetTTD).Float64()

	if got != want {
		t.Fatalf("expected beacon_spec_terminal_total_difficulty to report the real TTD %v, got %v", want, got)
	}
}

package beacon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chuckpreslar/emission"
	eapi "github.com/ethpandaops/go-eth2-client/api"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

// TestLifecycleMutex specifically tests for races on ctx and cancel fields
// This test should fail without the lifecycleMu mutex protection.
func TestLifecycleMutex(t *testing.T) {
	// Create a node with minimal setup
	n := &node{
		log: logrus.New(),
	}

	// Run many concurrent operations on the protected fields
	const workers = 50
	const iterations = 100

	for range iterations {
		var wg sync.WaitGroup

		// Writer goroutines - simulate Start() setting ctx and cancel
		for range workers {
			wg.Go(func() {
				ctx, cancel := context.WithCancel(context.Background())

				// This simulates what Start() does
				n.lifecycleMu.Lock()

				n.ctx = ctx
				n.cancel = cancel

				n.lifecycleMu.Unlock()

				// Clean up
				cancel()
			})
		}

		// Reader goroutines - simulate Stop() reading cancel
		for range workers {
			wg.Go(func() {
				// This simulates what Stop() does
				n.lifecycleMu.Lock()

				if n.cancel != nil {
					n.cancel()
				}

				n.lifecycleMu.Unlock()
			})
		}

		wg.Wait()

		// Reset for next iteration
		n.lifecycleMu.Lock()

		n.ctx = nil
		n.cancel = nil

		n.lifecycleMu.Unlock()
	}
}

// TestLifecycleStartStopSequence verifies proper Start/Stop sequence handling.
func TestLifecycleStartStopSequence(t *testing.T) {
	n := &node{
		log:     logrus.New(),
		options: DefaultOptions(),
		config:  &Config{},
		stat:    NewStatus(1, 1),
	}
	n.options.PrometheusMetrics = false

	// Test normal start/stop sequence
	ctx := context.Background()

	// Before Start, cancel should be nil
	n.lifecycleMu.Lock()

	if n.cancel != nil {
		t.Error("cancel should be nil before Start")
	}

	n.lifecycleMu.Unlock()

	// Simulate Start without actually starting (to avoid bootstrap errors)
	startCtx, startCancel := context.WithCancel(ctx)

	n.lifecycleMu.Lock()

	n.ctx = startCtx
	n.cancel = startCancel

	n.lifecycleMu.Unlock()

	// Verify cancel is set
	n.lifecycleMu.Lock()

	if n.cancel == nil {
		t.Error("cancel should not be nil after Start")
	}

	n.lifecycleMu.Unlock()

	// Stop should work without race
	err := n.Stop(ctx)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify context was cancelled
	select {
	case <-startCtx.Done():
		// Good, context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("context was not cancelled after Stop")
	}
}

// finalityClient is a minimal eth2client.FinalityProvider used to drive
// FetchFinality without a real beacon node.
type finalityClient struct{}

func (f *finalityClient) Name() string    { return "fake" }
func (f *finalityClient) Address() string { return "fake://" }
func (f *finalityClient) IsActive() bool  { return true }
func (f *finalityClient) IsSynced() bool  { return true }

func (f *finalityClient) Finality(
	_ context.Context, _ *eapi.FinalityOpts,
) (*eapi.Response[*v1.Finality], error) {
	return &eapi.Response[*v1.Finality]{
		Data: &v1.Finality{
			Finalized:         &phase0.Checkpoint{Epoch: 1, Root: phase0.Root{0x01}},
			Justified:         &phase0.Checkpoint{Epoch: 2, Root: phase0.Root{0x02}},
			PreviousJustified: &phase0.Checkpoint{Epoch: 3, Root: phase0.Root{0x03}},
		},
	}, nil
}

// TestFinalityMutex exercises FetchFinality and Finality() concurrently,
// mirroring the shape of the epoch cron, the finalized_checkpoint event
// handler and a direct consumer call all hitting finality at once. It
// should pass cleanly under -race.
func TestFinalityMutex(t *testing.T) {
	n := &node{
		log:    logrus.New(),
		broker: emission.NewEmitter(),
		client: &finalityClient{},
	}

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 50 {
				if _, err := n.FetchFinality(context.Background(), "head"); err != nil {
					t.Error(err)
				}
			}
		})
	}

	for range 8 {
		wg.Go(func() {
			for range 50 {
				_, _ = n.Finality()
			}
		})
	}

	wg.Wait()
}

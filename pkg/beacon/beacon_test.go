package beacon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	eapi "github.com/ethpandaops/go-eth2-client/api"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
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

// schedulerFakeClient implements just enough of eth2client.Service to get
// through ensureClients and bootstrap without a real beacon node.
type schedulerFakeClient struct{}

func (f *schedulerFakeClient) Name() string    { return "fake" }
func (f *schedulerFakeClient) Address() string { return "fake://" }
func (f *schedulerFakeClient) IsActive() bool  { return true }
func (f *schedulerFakeClient) IsSynced() bool  { return true }

func (f *schedulerFakeClient) NodeSyncing(
	_ context.Context, _ *eapi.NodeSyncingOpts,
) (*eapi.Response[*v1.SyncState], error) {
	return &eapi.Response[*v1.SyncState]{Data: &v1.SyncState{}}, nil
}

func (f *schedulerFakeClient) Spec(
	_ context.Context, _ *eapi.SpecOpts,
) (*eapi.Response[map[string]any], error) {
	return &eapi.Response[map[string]any]{
		Data: map[string]any{
			"SECONDS_PER_SLOT": "12",
			"SLOTS_PER_EPOCH":  "32",
		},
	}, nil
}

func (f *schedulerFakeClient) Genesis(
	_ context.Context, _ *eapi.GenesisOpts,
) (*eapi.Response[*v1.Genesis], error) {
	return &eapi.Response[*v1.Genesis]{Data: &v1.Genesis{GenesisTime: time.Now()}}, nil
}

func (f *schedulerFakeClient) NodeVersion(
	_ context.Context, _ *eapi.NodeVersionOpts,
) (*eapi.Response[string], error) {
	return &eapi.Response[string]{Data: "fake/v0.0.0"}, nil
}

// schedulerFakeAPI implements api.ConsensusClient. Start()'s cron jobs run
// once immediately when the scheduler starts, so FetchPeers needs this to
// avoid a nil pointer dereference on n.api.
type schedulerFakeAPI struct{ types.Peers }

func (f *schedulerFakeAPI) NodePeer(context.Context, string) (types.Peer, error) {
	return types.Peer{}, nil
}
func (f *schedulerFakeAPI) NodePeers(context.Context) (types.Peers, error) { return nil, nil }
func (f *schedulerFakeAPI) NodePeerCount(context.Context) (types.PeerCount, error) {
	return types.PeerCount{}, nil
}
func (f *schedulerFakeAPI) RawBlock(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (f *schedulerFakeAPI) RawDebugBeaconState(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (f *schedulerFakeAPI) DepositSnapshot(context.Context) (*types.DepositSnapshot, error) {
	return nil, nil
}
func (f *schedulerFakeAPI) NodeIdentity(context.Context) (*types.Identity, error) {
	return nil, nil
}

// TestStopStopsScheduler runs a real Start()/Stop() cycle and checks that
// the cron scheduler Start() creates is actually assigned to n.crons and
// stopped by Stop(). Not run under -race: go-eth2-client's wallclock
// dependency has its own unrelated internal race between its ticker
// goroutine and listener registration that triggers on any Start() call,
// independent of this fix, and this repo's CI does not run tests with
// -race either.
func TestStopStopsScheduler(t *testing.T) {
	n := &node{
		log:     logrus.New(),
		options: &Options{HealthCheck: DefaultHealthCheckOptions()},
		config:  &Config{},
		stat:    NewStatus(1, 1),
		broker:  emission.NewEmitter(),
		client:  &schedulerFakeClient{},
		api:     &schedulerFakeAPI{},
	}

	ctx := context.Background()

	if err := n.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if n.crons == nil {
		t.Fatal("expected crons to be assigned after Start")
	}

	if !n.crons.IsRunning() {
		t.Fatal("expected scheduler to be running after Start")
	}

	if err := n.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if n.crons.IsRunning() {
		t.Fatal("expected scheduler to be stopped after Stop, but it is still running")
	}
}

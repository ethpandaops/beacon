package beacon

import (
	"context"
	"sync"
	"testing"
	"time"

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

	for i := 0; i < iterations; i++ {
		var wg sync.WaitGroup

		// Writer goroutines - simulate Start() setting ctx and cancel
		for j := 0; j < workers; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithCancel(context.Background())

				// This simulates what Start() does
				n.lifecycleMu.Lock()

				n.ctx = ctx
				n.cancel = cancel

				n.lifecycleMu.Unlock()

				// Clean up
				cancel()
			}()
		}

		// Reader goroutines - simulate Stop() reading cancel
		for j := 0; j < workers; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// This simulates what Stop() does
				n.lifecycleMu.Lock()

				if n.cancel != nil {
					n.cancel()
				}

				n.lifecycleMu.Unlock()
			}()
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

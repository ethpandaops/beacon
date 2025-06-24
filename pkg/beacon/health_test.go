package beacon

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// TestHealthConcurrentAccess tests that Health methods are thread-safe.
func TestHealthConcurrentAccess(t *testing.T) {
	h := NewHealth(3, 3)

	// Number of goroutines to spawn
	numGoroutines := 100
	numOperations := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations per goroutine

	// Spawn goroutines that call RecordSuccess
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				h.RecordSuccess()
			}
		}()
	}

	// Spawn goroutines that call RecordFail
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				h.RecordFail(errors.New("test error"))
			}
		}()
	}

	// Spawn goroutines that read health status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = h.Healthy()
				_ = h.SuccessTotal()
				_ = h.FailedTotal()
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify final counts are correct
	expectedTotal := uint64(numGoroutines * numOperations)
	successTotal := h.SuccessTotal()
	failTotal := h.FailedTotal()

	if successTotal != expectedTotal {
		t.Errorf("Expected success total %d, got %d", expectedTotal, successTotal)
	}

	if failTotal != expectedTotal {
		t.Errorf("Expected fail total %d, got %d", expectedTotal, failTotal)
	}
}

// TestHealthRaceCondition specifically tests for race conditions using parallel test execution.
func TestHealthRaceCondition(t *testing.T) {
	h := NewHealth(2, 2)

	// Run parallel tests to trigger race detector
	t.Run("parallel", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 100; i++ {
				h.RecordSuccess()
				time.Sleep(time.Microsecond)
			}
		})

		t.Run("fail", func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 100; i++ {
				h.RecordFail(errors.New("test"))
				time.Sleep(time.Microsecond)
			}
		})

		t.Run("read", func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 100; i++ {
				_ = h.Healthy()
				_ = h.SuccessTotal()
				_ = h.FailedTotal()
				time.Sleep(time.Microsecond)
			}
		})
	})
}

// TestHealthThresholds tests that health status changes correctly based on thresholds.
func TestHealthThresholds(t *testing.T) {
	h := NewHealth(3, 2)

	// Initially should not be healthy
	if h.Healthy() {
		t.Error("Expected initial state to be unhealthy")
	}

	// Record 2 successes - still not healthy (threshold is 3)
	h.RecordSuccess()
	h.RecordSuccess()
	if h.Healthy() {
		t.Error("Expected unhealthy after 2 successes (threshold is 3)")
	}

	// Record 3rd success - should be healthy now
	h.RecordSuccess()
	if !h.Healthy() {
		t.Error("Expected healthy after 3 successes")
	}

	// Record 1 failure - should remain healthy (threshold is 2)
	h.RecordFail(errors.New("test"))
	if !h.Healthy() {
		t.Error("Expected healthy after 1 failure (threshold is 2)")
	}

	// Record 2nd failure - should be unhealthy now
	h.RecordFail(errors.New("test"))
	if h.Healthy() {
		t.Error("Expected unhealthy after 2 failures")
	}
}

// TestHealthCounters tests that success and failure counters work correctly.
func TestHealthCounters(t *testing.T) {
	h := NewHealth(1, 1)

	// Test success counter
	for i := 1; i <= 5; i++ {
		h.RecordSuccess()
		if h.SuccessTotal() != uint64(i) {
			t.Errorf("Expected success total %d, got %d", i, h.SuccessTotal())
		}
	}

	// Test failure counter
	for i := 1; i <= 5; i++ {
		h.RecordFail(errors.New("test"))
		if h.FailedTotal() != uint64(i) {
			t.Errorf("Expected fail total %d, got %d", i, h.FailedTotal())
		}
	}

	// Verify totals remain correct
	if h.SuccessTotal() != 5 {
		t.Errorf("Expected final success total 5, got %d", h.SuccessTotal())
	}
	if h.FailedTotal() != 5 {
		t.Errorf("Expected final fail total 5, got %d", h.FailedTotal())
	}
}

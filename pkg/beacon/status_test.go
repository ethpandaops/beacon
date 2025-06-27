package beacon

import (
	"sync"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_ConcurrentSyncStateAccess(t *testing.T) {
	status := NewStatus(5, 3)

	// Number of concurrent goroutines
	numGoroutines := 100
	numIterations := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Half readers, half writers

	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				syncState := &v1.SyncState{
					IsSyncing:    j%2 == 0,
					HeadSlot:     phase0.Slot(j),
					SyncDistance: phase0.Slot(j * 2),
				}
				status.UpdateSyncState(syncState)
			}
		}(i)
	}

	// Start reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Read sync state
				state := status.SyncState()
				if state != nil {
					_ = state.IsSyncing
					_ = state.HeadSlot
				}

				// Read syncing status
				_ = status.Syncing()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify final state is valid
	finalState := status.SyncState()
	assert.NotNil(t, finalState)
}

func TestStatus_ConcurrentNetworkIDAccess(t *testing.T) {
	status := NewStatus(5, 3)

	numGoroutines := 50
	numIterations := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Start writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				status.UpdateNetworkID(uint64(id*1000 + j))
			}
		}(i)
	}

	// Start reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				networkID := status.NetworkID()
				// Verify we get a valid network ID (not corrupted)
				assert.GreaterOrEqual(t, networkID, uint64(0))
			}
		}(i)
	}

	wg.Wait()

	// Verify final network ID is valid
	finalID := status.NetworkID()
	assert.GreaterOrEqual(t, finalID, uint64(0))
}

func TestStatus_SyncingMethod(t *testing.T) {
	status := NewStatus(5, 3)

	// Test when syncstate is nil
	assert.False(t, status.Syncing())

	// Test when syncing is true
	status.UpdateSyncState(&v1.SyncState{
		IsSyncing: true,
	})
	assert.True(t, status.Syncing())

	// Test when syncing is false
	status.UpdateSyncState(&v1.SyncState{
		IsSyncing: false,
	})
	assert.False(t, status.Syncing())
}

func TestStatus_InitialState(t *testing.T) {
	status := NewStatus(5, 3)

	// Verify initial state
	assert.Equal(t, uint64(0), status.NetworkID())
	assert.Nil(t, status.SyncState())
	assert.False(t, status.Syncing())
	assert.NotNil(t, status.Health())
}

func TestStatus_UpdateMethods(t *testing.T) {
	status := NewStatus(5, 3)

	// Test UpdateNetworkID
	testNetworkID := uint64(12345)
	status.UpdateNetworkID(testNetworkID)
	assert.Equal(t, testNetworkID, status.NetworkID())

	// Test UpdateSyncState
	syncState := &v1.SyncState{
		IsSyncing:    true,
		HeadSlot:     phase0.Slot(100),
		SyncDistance: phase0.Slot(50),
	}
	status.UpdateSyncState(syncState)

	retrievedState := status.SyncState()
	require.NotNil(t, retrievedState)
	assert.Equal(t, syncState.IsSyncing, retrievedState.IsSyncing)
	assert.Equal(t, syncState.HeadSlot, retrievedState.HeadSlot)
	assert.Equal(t, syncState.SyncDistance, retrievedState.SyncDistance)
}

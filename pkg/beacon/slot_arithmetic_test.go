package beacon

import "testing"

func TestPreviousSlotStateID(t *testing.T) {
	tests := []struct {
		name       string
		slotNumber uint64
		expectedID string
		expectedOK bool
	}{
		{
			name:       "slot 0 has no previous slot",
			slotNumber: 0,
			expectedID: "",
			expectedOK: false,
		},
		{
			name:       "slot 1 returns slot 0",
			slotNumber: 1,
			expectedID: "0",
			expectedOK: true,
		},
		{
			name:       "an ordinary slot returns the slot before it",
			slotNumber: 12345,
			expectedID: "12344",
			expectedOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := previousSlotStateID(tt.slotNumber)
			if ok != tt.expectedOK {
				t.Fatalf("expected ok=%v, got %v", tt.expectedOK, ok)
			}

			if id != tt.expectedID {
				t.Fatalf("expected id=%q, got %q", tt.expectedID, id)
			}
		})
	}
}

func TestBlockTooOldForProposerDelay(t *testing.T) {
	tests := []struct {
		name            string
		currSlotNumber  uint64
		blockSlotNumber uint64
		expected        bool
	}{
		{
			name:            "block at the current slot is not too old",
			currSlotNumber:  1000,
			blockSlotNumber: 1000,
			expected:        false,
		},
		{
			name:            "block 1 slot in the future is not too old",
			currSlotNumber:  1000,
			blockSlotNumber: 1001,
			expected:        false,
		},
		{
			name:            "block far in the future is not too old",
			currSlotNumber:  1000,
			blockSlotNumber: 5000,
			expected:        false,
		},
		{
			name:            "block 2 slots in the past is not too old",
			currSlotNumber:  1000,
			blockSlotNumber: 998,
			expected:        false,
		},
		{
			name:            "block 3 slots in the past is too old",
			currSlotNumber:  1000,
			blockSlotNumber: 997,
			expected:        true,
		},
		{
			name:            "block at slot 0 with current slot far ahead is too old",
			currSlotNumber:  1000,
			blockSlotNumber: 0,
			expected:        true,
		},
		{
			name:            "current slot at 0 with a future block is not too old",
			currSlotNumber:  0,
			blockSlotNumber: 5,
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockTooOldForProposerDelay(tt.currSlotNumber, tt.blockSlotNumber)
			if got != tt.expected {
				t.Fatalf("blockTooOldForProposerDelay(%d, %d) = %v, want %v",
					tt.currSlotNumber, tt.blockSlotNumber, got, tt.expected)
			}
		})
	}
}

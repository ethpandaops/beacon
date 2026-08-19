package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSpec_TerminalTotalDifficulty(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "decimal string parses correctly",
			value:    "58750000000000000000000",
			expected: "58750000000000000000000",
		},
		{
			name:     "zero parses correctly",
			value:    "0",
			expected: "0",
		},
		{
			name:     "hex-encoded byte slice does not panic and leaves the zero value",
			value:    []byte{0xc7, 0x0d, 0x80, 0x8a, 0x12, 0x8d, 0x73, 0x80},
			expected: "0",
		},
		{
			name:     "non-numeric string does not panic and leaves the zero value",
			value:    "not-a-number",
			expected: "0",
		},
		{
			name:     "empty string does not panic and leaves the zero value",
			value:    "",
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				spec := NewSpec(map[string]any{"TERMINAL_TOTAL_DIFFICULTY": tt.value})
				assert.Equal(t, tt.expected, spec.TerminalTotalDifficulty.String())
			})
		})
	}

	t.Run("field absent leaves the zero value", func(t *testing.T) {
		spec := NewSpec(map[string]any{})
		assert.Equal(t, big.NewInt(0).String(), spec.TerminalTotalDifficulty.String())
	})
}

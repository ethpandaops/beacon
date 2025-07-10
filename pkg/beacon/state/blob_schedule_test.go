package state

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
)

func TestBlobSchedule_GetMaxBlobsPerBlock(t *testing.T) {
	tests := []struct {
		name         string
		schedule     BlobSchedule
		epoch        phase0.Epoch
		expectedBlob uint64
	}{
		{
			name:         "empty schedule returns 0",
			schedule:     BlobSchedule{},
			epoch:        phase0.Epoch(100),
			expectedBlob: 0,
		},
		{
			name: "epoch before any schedule entry returns minimum",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
			},
			epoch:        phase0.Epoch(50),
			expectedBlob: 6,
		},
		{
			name: "epoch exactly matches first entry",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
			},
			epoch:        phase0.Epoch(100),
			expectedBlob: 6,
		},
		{
			name: "epoch exactly matches second entry",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
			},
			epoch:        phase0.Epoch(200),
			expectedBlob: 9,
		},
		{
			name: "epoch between entries uses earlier entry",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
			},
			epoch:        phase0.Epoch(150),
			expectedBlob: 6,
		},
		{
			name: "epoch after all entries uses latest entry",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
			},
			epoch:        phase0.Epoch(300),
			expectedBlob: 9,
		},
		{
			name: "unordered schedule entries are handled correctly",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
				{Epoch: phase0.Epoch(300), MaxBlobsPerBlock: 12},
			},
			epoch:        phase0.Epoch(250),
			expectedBlob: 9,
		},
		{
			name: "fulu testnet example - deneb epoch",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(512), MaxBlobsPerBlock: 12},
				{Epoch: phase0.Epoch(768), MaxBlobsPerBlock: 15},
				{Epoch: phase0.Epoch(1024), MaxBlobsPerBlock: 18},
				{Epoch: phase0.Epoch(1280), MaxBlobsPerBlock: 9},
				{Epoch: phase0.Epoch(1584), MaxBlobsPerBlock: 20},
			},
			epoch:        phase0.Epoch(600),
			expectedBlob: 12,
		},
		{
			name: "fulu testnet example - electra epoch",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(512), MaxBlobsPerBlock: 12},
				{Epoch: phase0.Epoch(768), MaxBlobsPerBlock: 15},
				{Epoch: phase0.Epoch(1024), MaxBlobsPerBlock: 18},
				{Epoch: phase0.Epoch(1280), MaxBlobsPerBlock: 9},
				{Epoch: phase0.Epoch(1584), MaxBlobsPerBlock: 20},
			},
			epoch:        phase0.Epoch(1300),
			expectedBlob: 9,
		},
		{
			name: "fulu testnet example - latest epoch",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(512), MaxBlobsPerBlock: 12},
				{Epoch: phase0.Epoch(768), MaxBlobsPerBlock: 15},
				{Epoch: phase0.Epoch(1024), MaxBlobsPerBlock: 18},
				{Epoch: phase0.Epoch(1280), MaxBlobsPerBlock: 9},
				{Epoch: phase0.Epoch(1584), MaxBlobsPerBlock: 20},
			},
			epoch:        phase0.Epoch(2000),
			expectedBlob: 20,
		},
		{
			name: "single entry schedule",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
			},
			epoch:        phase0.Epoch(150),
			expectedBlob: 6,
		},
		{
			name: "single entry schedule before epoch",
			schedule: BlobSchedule{
				{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
			},
			epoch:        phase0.Epoch(50),
			expectedBlob: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schedule.GetMaxBlobsPerBlock(tt.epoch)
			assert.Equal(t, tt.expectedBlob, result, "Expected %d blobs for epoch %d", tt.expectedBlob, tt.epoch)
		})
	}
}

func TestSpec_GetMaxBlobsPerBlock(t *testing.T) {
	spec := Spec{
		BlobSchedule: BlobSchedule{
			{Epoch: phase0.Epoch(100), MaxBlobsPerBlock: 6},
			{Epoch: phase0.Epoch(200), MaxBlobsPerBlock: 9},
		},
	}

	result := spec.GetMaxBlobsPerBlock(phase0.Epoch(150))
	assert.Equal(t, uint64(6), result, "Spec.GetMaxBlobsPerBlock should delegate to BlobSchedule.GetMaxBlobsPerBlock")
}

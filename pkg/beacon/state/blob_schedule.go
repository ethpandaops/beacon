package state

import (
	"sort"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// BlobScheduleEntry represents a single entry in the BLOB_SCHEDULE configuration.
type BlobScheduleEntry struct {
	Epoch            phase0.Epoch `json:"EPOCH,string"`
	MaxBlobsPerBlock uint64       `json:"MAX_BLOBS_PER_BLOCK,string"`
}

// BlobSchedule represents the BLOB_SCHEDULE configuration.
type BlobSchedule []BlobScheduleEntry

// GetMaxBlobsPerBlock returns the maximum number of blobs that can be included in a block for a given epoch.
func (bs BlobSchedule) GetMaxBlobsPerBlock(epoch phase0.Epoch) uint64 {
	if len(bs) == 0 {
		return 0
	}

	// Sort by epoch in descending order to find the most recent applicable entry.
	sorted := make(BlobSchedule, len(bs))
	copy(sorted, bs)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Epoch > sorted[j].Epoch
	})

	// Find the first entry where epoch >= entry.Epoch.
	for _, entry := range sorted {
		if epoch >= entry.Epoch {
			return entry.MaxBlobsPerBlock
		}
	}

	// If no entry is found, return the minimum value from all entries.
	minBlobs := sorted[0].MaxBlobsPerBlock
	for _, entry := range sorted {
		if entry.MaxBlobsPerBlock < minBlobs {
			minBlobs = entry.MaxBlobsPerBlock
		}
	}

	return minBlobs
}

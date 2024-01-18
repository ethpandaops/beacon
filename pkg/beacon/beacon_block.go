package beacon

import (
	"github.com/attestantio/go-eth2-client/spec"
)

// GetDepositCountsFromBeaconBlock returns the number of deposits in a beacon block
func GetDepositCountsFromBeaconBlock(block *spec.VersionedSignedBeaconBlock) int {
	deposits, err := block.Deposits()
	if err == nil {
		return len(deposits)
	}

	return 0
}

// GetVoluntaryExitsFromBeaconBlock returns the number of voluntary exits in a beacon block
func GetVoluntaryExitsFromBeaconBlock(block *spec.VersionedSignedBeaconBlock) int {
	exits, err := block.VoluntaryExits()
	if err == nil {
		return len(exits)
	}

	return 0
}

// GetTransactionsCountFromBeaconBlock returns the number of transactions in a beacon block
func GetTransactionsCountFromBeaconBlock(block *spec.VersionedSignedBeaconBlock) int {
	transactions, err := block.ExecutionTransactions()
	if err == nil {
		return len(transactions)
	}

	return 0
}

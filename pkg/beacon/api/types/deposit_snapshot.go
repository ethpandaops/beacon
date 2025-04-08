package types

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// DepositSnapshot represents an EIP-4881 deposit snapshot.
type DepositSnapshot struct {
	Finalized            []phase0.Root `json:"finalized"`
	DepositRoot          phase0.Root   `json:"deposit_root"`
	DepositCount         uint64        `json:"deposit_count"`
	ExecutionBlockHash   phase0.Root   `json:"execution_block_hash"`
	ExecutionBlockHeight uint64        `json:"execution_block_height"`
}

type DepositSnapshotJSON struct {
	Finalized            []string `json:"finalized"`
	DepositRoot          string   `json:"deposit_root"`
	DepositCount         uint64   `json:"deposit_count,string"`
	ExecutionBlockHash   string   `json:"execution_block_hash"`
	ExecutionBlockHeight uint64   `json:"execution_block_height,string"`
}

func (d *DepositSnapshot) MarshalJSON() ([]byte, error) {
	finalized := make([]string, len(d.Finalized))
	for i := range d.Finalized {
		finalized[i] = d.Finalized[i].String()
	}

	return json.Marshal(&DepositSnapshotJSON{
		Finalized:            finalized,
		DepositRoot:          d.DepositRoot.String(),
		DepositCount:         d.DepositCount,
		ExecutionBlockHash:   d.ExecutionBlockHash.String(),
		ExecutionBlockHeight: d.ExecutionBlockHeight,
	})
}

func (d *DepositSnapshot) UnmarshalJSON(input []byte) error {
	var err error

	var depositSnapshotJSON DepositSnapshotJSON
	if err = json.Unmarshal(input, &depositSnapshotJSON); err != nil {
		return err
	}

	d.Finalized = make([]phase0.Root, len(depositSnapshotJSON.Finalized))

	for i := range depositSnapshotJSON.Finalized {
		root, rootErr := hex.DecodeString(strings.TrimPrefix(depositSnapshotJSON.Finalized[i], "0x"))
		if rootErr != nil {
			return rootErr
		}

		if len(root) != 32 {
			return errors.New("incorrect length for root")
		}

		copy(d.Finalized[i][:], root)
	}

	depositRoot, err := hex.DecodeString(strings.TrimPrefix(depositSnapshotJSON.DepositRoot, "0x"))
	if err != nil {
		return err
	}

	if len(depositRoot) != 32 {
		return errors.New("incorrect length for deposit root")
	}

	copy(d.DepositRoot[:], depositRoot)

	d.DepositCount = depositSnapshotJSON.DepositCount

	executionBlockHash, err := hex.DecodeString(strings.TrimPrefix(depositSnapshotJSON.ExecutionBlockHash, "0x"))
	if err != nil {
		return err
	}

	if len(executionBlockHash) != 32 {
		return errors.New("incorrect length for execution block hash")
	}

	copy(d.ExecutionBlockHash[:], executionBlockHash)

	d.ExecutionBlockHeight = depositSnapshotJSON.ExecutionBlockHeight

	return nil
}

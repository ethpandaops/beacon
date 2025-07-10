package state

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	sp "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/spf13/cast"
)

// Spec represents the state of the spec.
type Spec struct {
	PresetBase string `json:"PRESET_BASE"`
	ConfigName string `json:"CONFIG_NAME"`

	DepositChainID         uint64 `json:"DEPOSIT_CHAIN_ID,string"`
	DepositContractAddress string `json:"DEPOSIT_CONTRACT_ADDRESS"`

	SafeSlotsToUpdateJustified phase0.Slot `json:"SAFE_SLOTS_TO_UPDATE_JUSTIFIED,string"`
	SlotsPerEpoch              phase0.Slot `json:"SLOTS_PER_EPOCH,string"`

	EpochsPerSyncCommitteePeriod phase0.Epoch `json:"EPOCHS_PER_SYNC_COMMITTEE_PERIOD,string"`
	MinSyncCommitteeParticipants uint64       `json:"MIN_SYNC_COMMITTEE_PARTICIPANTS,string"`
	TargetCommitteeSize          uint64       `json:"TARGET_COMMITTEE_SIZE,string"`
	SyncCommitteeSize            uint64       `json:"SYNC_COMMITTEE_SIZE,string"`

	TerminalBlockHashActivationEpoch phase0.Epoch `json:"TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH,string"`
	TerminalTotalDifficulty          big.Int      `json:"-"`

	MaxValidatorsPerCommittee uint64      `json:"MAX_VALIDATORS_PER_COMMITTEE,string"`
	BaseRewardFactor          uint64      `json:"BASE_REWARD_FACTOR,string"`
	EffectiveBalanceIncrement phase0.Gwei `json:"EFFECTIVE_BALANCE_INCREMENT,string"`
	MaxEffectiveBalance       phase0.Gwei `json:"MAX_EFFECTIVE_BALANCE,string"`
	MinDepositAmount          phase0.Gwei `json:"MIN_DEPOSIT_AMOUNT,string"`
	MaxAttestations           uint64      `json:"MAX_ATTESTATIONS,string"`

	SecondsPerEth1Block            StringerDuration `json:"SECONDS_PER_ETH1_BLOCK,string"`
	GenesisDelay                   StringerDuration `json:"GENESIS_DELAY,string"`
	SecondsPerSlot                 StringerDuration `json:"SECONDS_PER_SLOT,string"`
	MaxDeposits                    uint64           `json:"MAX_DEPOSITS,string"`
	MinGenesisActiveValidatorCount uint64           `json:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT,string"`
	Eth1FollowDistance             uint64           `json:"ETH1_FOLLOW_DISTANCE,string"`

	ForkEpochs   ForkEpochs   `json:"-"`
	BlobSchedule BlobSchedule `json:"BLOB_SCHEDULE"`
}

// NewSpec creates a new spec instance.
//
//nolint:gocyclo // existing.
func NewSpec(data map[string]interface{}) Spec {
	spec := Spec{
		ForkEpochs: ForkEpochs{},
	}

	if safeSlotsToUpdateJustified, exists := data["SAFE_SLOTS_TO_UPDATE_JUSTIFIED"]; exists {
		spec.SafeSlotsToUpdateJustified = phase0.Slot(cast.ToUint64(safeSlotsToUpdateJustified))
	}

	if depositChainID, exists := data["DEPOSIT_CHAIN_ID"]; exists {
		spec.DepositChainID = cast.ToUint64(depositChainID)
	}

	if depositContractAddress, exists := data["DEPOSIT_CONTRACT_ADDRESS"]; exists {
		spec.DepositContractAddress = fmt.Sprintf("%#x", cast.ToString(depositContractAddress))
	}

	if configName, exists := data["CONFIG_NAME"]; exists {
		spec.ConfigName = cast.ToString(configName)
	}

	if maxValidatorsPerCommittee, exists := data["MAX_VALIDATORS_PER_COMMITTEE"]; exists {
		spec.MaxValidatorsPerCommittee = cast.ToUint64(maxValidatorsPerCommittee)
	}

	if secondsPerEth1Block, exists := data["SECONDS_PER_ETH1_BLOCK"]; exists {
		spec.SecondsPerEth1Block = StringerDuration(cast.ToDuration(secondsPerEth1Block))
	}

	if baseRewardFactor, exists := data["BASE_REWARD_FACTOR"]; exists {
		spec.BaseRewardFactor = cast.ToUint64(baseRewardFactor)
	}

	if epochsPerSyncComitteePeriod, exists := data["EPOCHS_PER_SYNC_COMMITTEE_PERIOD"]; exists {
		spec.EpochsPerSyncCommitteePeriod = phase0.Epoch(cast.ToUint64(epochsPerSyncComitteePeriod))
	}

	if effectiveBalanceIncrement, exists := data["EFFECTIVE_BALANCE_INCREMENT"]; exists {
		spec.EffectiveBalanceIncrement = phase0.Gwei(cast.ToUint64(effectiveBalanceIncrement))
	}

	if maxAttestations, exists := data["MAX_ATTESTATIONS"]; exists {
		spec.MaxAttestations = cast.ToUint64(maxAttestations)
	}

	if minSyncCommitteeParticipants, exists := data["MIN_SYNC_COMMITTEE_PARTICIPANTS"]; exists {
		spec.MinSyncCommitteeParticipants = cast.ToUint64(minSyncCommitteeParticipants)
	}

	if genesisDelay, exists := data["GENESIS_DELAY"]; exists {
		spec.GenesisDelay = StringerDuration(cast.ToDuration(genesisDelay))
	}

	if secondsPerSlot, exists := data["SECONDS_PER_SLOT"]; exists {
		spec.SecondsPerSlot = StringerDuration(cast.ToDuration(secondsPerSlot))
	}

	if maxEffectiveBalance, exists := data["MAX_EFFECTIVE_BALANCE"]; exists {
		spec.MaxEffectiveBalance = phase0.Gwei(cast.ToUint64(maxEffectiveBalance))
	}

	if terminalTotalDifficulty, exists := data["TERMINAL_TOTAL_DIFFICULTY"]; exists {
		ttd := cast.ToString(fmt.Sprintf("%v", terminalTotalDifficulty))

		casted, _ := (*big.NewInt(0)).SetString(ttd, 10)
		spec.TerminalTotalDifficulty = *casted
	}

	if maxDeposits, exists := data["MAX_DEPOSITS"]; exists {
		spec.MaxDeposits = cast.ToUint64(maxDeposits)
	}

	if minGenesisActiveValidatorCount, exists := data["MIN_GENESIS_ACTIVE_VALIDATOR_COUNT"]; exists {
		spec.MinGenesisActiveValidatorCount = cast.ToUint64(minGenesisActiveValidatorCount)
	}

	if targetCommitteeSize, exists := data["TARGET_COMMITTEE_SIZE"]; exists {
		spec.TargetCommitteeSize = cast.ToUint64(targetCommitteeSize)
	}

	if syncCommitteeSize, exists := data["SYNC_COMMITTEE_SIZE"]; exists {
		spec.SyncCommitteeSize = cast.ToUint64(syncCommitteeSize)
	}

	if eth1FollowDistance, exists := data["ETH1_FOLLOW_DISTANCE"]; exists {
		spec.Eth1FollowDistance = cast.ToUint64(eth1FollowDistance)
	}

	if terminalBlockHashActivationEpoch, exists := data["TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH"]; exists {
		spec.TerminalBlockHashActivationEpoch = phase0.Epoch(cast.ToUint64(terminalBlockHashActivationEpoch))
	}

	if minDepositAmount, exists := data["MIN_DEPOSIT_AMOUNT"]; exists {
		spec.MinDepositAmount = phase0.Gwei(cast.ToUint64(minDepositAmount))
	}

	if slotsPerEpoch, exists := data["SLOTS_PER_EPOCH"]; exists {
		spec.SlotsPerEpoch = phase0.Slot(cast.ToUint64(slotsPerEpoch))
	}

	if presetBase, exists := data["PRESET_BASE"]; exists {
		spec.PresetBase = cast.ToString(presetBase)
	}

	forkEpochs := make(map[string]phase0.Epoch)
	forkVersions := make(map[string]string)

	forkEpochs["GENESIS"] = 0

	for k, v := range data {
		if strings.Contains(k, "_FORK_EPOCH") {
			forkName := strings.ReplaceAll(k, "_FORK_EPOCH", "")

			forkEpochs[forkName] = phase0.Epoch(cast.ToUint64(v))
		}

		if strings.Contains(k, "_FORK_VERSION") {
			forkName := strings.ReplaceAll(k, "_FORK_VERSION", "")

			forkVersions[forkName] = fmt.Sprintf("%#x", v)
		}
	}

	for k, v := range forkEpochs {
		version := ""
		if v, exists := forkVersions[k]; exists {
			version = v
		}

		// Convert the name to a DataVersion.
		dataVersion, err := dataVersionFromString(k)
		if err != nil {
			continue
		}

		spec.ForkEpochs = append(spec.ForkEpochs, &ForkEpoch{
			Epoch:   v,
			Name:    dataVersion,
			Version: version,
		})
	}

	if blobSchedule, exists := data["BLOB_SCHEDULE"]; exists {
		if scheduleData, ok := blobSchedule.([]interface{}); ok {
			spec.BlobSchedule = make(BlobSchedule, len(scheduleData))

			for i, entry := range scheduleData {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					spec.BlobSchedule[i] = BlobScheduleEntry{
						Epoch:            phase0.Epoch(cast.ToUint64(entryMap["EPOCH"])),
						MaxBlobsPerBlock: cast.ToUint64(entryMap["MAX_BLOBS_PER_BLOCK"]),
					}
				}
			}
		}
	}

	return spec
}

// Validate performs basic validation of the spec.
func (s *Spec) Validate() error {
	return nil
}

// GetMaxBlobsPerBlock returns the maximum number of blobs that can be included in a block for a given epoch.
func (s *Spec) GetMaxBlobsPerBlock(epoch phase0.Epoch) uint64 {
	return s.BlobSchedule.GetMaxBlobsPerBlock(epoch)
}

func dataVersionFromString(name string) (sp.DataVersion, error) {
	var v sp.DataVersion
	if err := json.Unmarshal([]byte(fmt.Sprintf("\"%s\"", name)), &v); err != nil {
		return sp.DataVersionUnknown, err
	}

	return v, nil
}

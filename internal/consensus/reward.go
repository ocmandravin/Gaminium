package consensus

import "github.com/ocamndravin/gaminium/config"

// BlockReward returns the mining reward in Minium for a given block height.
func BlockReward(height int64) int64 {
	return config.HalvingSchedule(height)
}

// TotalIssuedSupply calculates total GMN issued up to a given block height (in Minium).
func TotalIssuedSupply(height int64) int64 {
	if height <= 0 {
		return 0
	}

	total := int64(0)
	halvingNum := int64(0)
	remaining := height

	for remaining > 0 && halvingNum < 64 {
		blocksInEra := config.HalvingInterval
		if remaining < blocksInEra {
			blocksInEra = remaining
		}

		rewardPerBlock := config.GenesisReward >> uint(halvingNum)
		total += rewardPerBlock * blocksInEra

		remaining -= blocksInEra
		halvingNum++
	}

	if total > config.TotalSupplyMinum {
		total = config.TotalSupplyMinum
	}
	return total
}

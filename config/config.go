package config

import "time"

const (
	// Identity
	Name   = "GAMINIUM"
	Ticker = "GMN"

	// Supply
	TotalSupply      = int64(51_200_000)    // GMN
	MiniumPerGMN     = int64(100_000_000)   // 1 GMN = 100,000,000 Minium
	TotalSupplyMinum = TotalSupply * MiniumPerGMN

	// Block parameters
	BlockTime         = 5 * time.Minute
	GenesisReward     = int64(50) * MiniumPerGMN // 50 GMN in Minium
	HalvingInterval   = int64(420_000)           // blocks
	MaxBlockSizeMin   = 1 * 1024 * 1024          // 1 MB
	MaxBlockSizeMax   = 8 * 1024 * 1024          // 8 MB
	Confirmations     = 6

	// Mempool
	MempoolMaxSize  = 300 * 1024 * 1024 // 300 MB
	MempoolExpiry   = 72 * time.Hour
	MinTransaction  = int64(1) // 1 Minium

	// Oracle
	OracleMinNodes       = 51
	OracleStakeRequired  = int64(10_000) * MiniumPerGMN // 10,000 GMN
	OracleMaxPerCountry  = 3
	OracleDeviationLimit = 0.10 // 10%
	OracleUpdateInterval = int64(2016)

	// Treasury allocation (basis points of fees)
	FeeMinerShare  = 70
	FeeTreasuryShare = 20
	FeeOracleShare = 10

	// Treasury composition targets (basis points)
	TreasuryUSDC = 40
	TreasuryBTC  = 30
	TreasuryETH  = 20
	TreasuryGMN  = 10

	// Floor defense
	FloorDefenseThreshold  = 0.02 // within 2% of floor
	FloorDefenseMaxEvent   = 0.10 // 10% treasury per event
	FloorDefenseMax30Days  = 0.30 // 30% treasury per 30 days

	// Governance
	VoteWeightLock3M  = 1
	VoteWeightLock1Y  = 4
	VoteWeightLock4Y  = 8
	MaxVotePerAddress = 0.05 // 5%

	// Multisig
	TreasuryMultisigM = 7
	TreasuryMultisigN = 12
	MaxMultisigKeys   = 15

	// Address prefix
	AddressPrefix = "GMN1"

	// Network
	DefaultPort     = 8333
	RPCPort         = 8332
	MaxPeers        = 125
	DNSSeedPrefix   = "seed"

	// AI confidence thresholds
	ConfidenceHigh   = 95.0
	ConfidenceMedium = 80.0
	ConfidenceLow    = 60.0

	// Author
	Author = "Ocamn Dravin"
)

// HalvingSchedule returns block reward in Minium for a given block height.
func HalvingSchedule(height int64) int64 {
	halvings := height / HalvingInterval
	if halvings >= 64 {
		return 0
	}
	reward := GenesisReward
	for i := int64(0); i < halvings; i++ {
		reward >>= 1
	}
	return reward
}

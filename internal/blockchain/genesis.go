package blockchain

import (
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// GenesisTimestamp is the GAMINIUM genesis block creation time.
// April 17 2026 — Ocamn Dravin
var GenesisTimestamp = time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC).Unix()

// GenesisBits is the initial difficulty target for block 0.
// Set to produce approximately 5-minute blocks on modest CPU hardware.
const GenesisBits = uint32(0x1e0fffff)

// GenesisExtraData is embedded in the genesis coinbase, inspired by Bitcoin's practice.
var GenesisExtraData = []byte("GAMINIUM/The Floor Holds/Ocamn Dravin/2026-04-17")

// GenesisBlock creates and returns the GAMINIUM genesis block.
// This function is deterministic — always produces the same genesis block.
func GenesisBlock(minerAddress string) *Block {
	coinbase := &CoinbaseTx{
		Height:       0,
		MinerAddress: minerAddress,
		Reward:       config.GenesisReward,
		ExtraData:    GenesisExtraData,
	}

	block := &Block{
		Header: BlockHeader{
			Version:    1,
			Height:     0,
			PrevHash:   crypto.Hash{}, // all zeros — no previous block
			Timestamp:  GenesisTimestamp,
			Bits:       GenesisBits,
			Nonce:      0,
			ExtraNonce: 0,
		},
		Transactions: nil,
		Coinbase:     coinbase,
	}

	// Compute and set the merkle root from coinbase only
	block.Header.MerkleRoot = block.ComputeMerkleRoot()

	// Set oracle data hash to genesis sentinel
	block.Header.OracleDataHash = crypto.HashMany(
		[]byte("genesis-oracle"),
		[]byte(config.Name),
		[]byte(config.Author),
	)

	return block
}

// HardcodedGenesisHash is the expected hash of the genesis block.
// Verified on startup to detect chain forks or corruption.
// This is set after mining the genesis block.
var HardcodedGenesisHash crypto.Hash

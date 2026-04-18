package tests

import (
	"testing"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/blockchain"
	"github.com/ocamndravin/gaminium/internal/consensus"
)

func TestGenesisBlock(t *testing.T) {
	genesis := blockchain.GenesisBlock("GMN1TESTADDRESS0000000000000000000000000000000000000000000000000000")
	if genesis == nil {
		t.Fatal("genesis block is nil")
	}
	if genesis.Header.Height != 0 {
		t.Errorf("genesis height: want 0, got %d", genesis.Header.Height)
	}
	if genesis.Coinbase == nil {
		t.Fatal("genesis coinbase is nil")
	}
	if genesis.Coinbase.Reward != config.GenesisReward {
		t.Errorf("genesis reward: want %d, got %d", config.GenesisReward, genesis.Coinbase.Reward)
	}
}

func TestChainInit(t *testing.T) {
	addr := "GMN1TESTADDRESS0000000000000000000000000000000000000000000000000000"
	genesis := blockchain.GenesisBlock(addr)
	chain, err := blockchain.NewChain(genesis)
	if err != nil {
		t.Fatalf("chain init: %v", err)
	}
	if chain.Height() != 0 {
		t.Errorf("initial height: want 0, got %d", chain.Height())
	}
	if chain.Genesis() == nil {
		t.Error("genesis should not be nil")
	}
}

func TestHalvingSchedule(t *testing.T) {
	cases := []struct {
		height int64
		want   int64
	}{
		{0, config.GenesisReward},
		{419999, config.GenesisReward},
		{420000, config.GenesisReward / 2},
		{840000, config.GenesisReward / 4},
		{1260000, config.GenesisReward / 8},
	}

	for _, tc := range cases {
		got := config.HalvingSchedule(tc.height)
		if got != tc.want {
			t.Errorf("height %d: want %d, got %d", tc.height, tc.want, got)
		}
	}
}

func TestBlockReward(t *testing.T) {
	reward := consensus.BlockReward(0)
	if reward != config.GenesisReward {
		t.Errorf("block 0 reward: want %d, got %d", config.GenesisReward, reward)
	}

	halvingReward := consensus.BlockReward(config.HalvingInterval)
	if halvingReward != config.GenesisReward/2 {
		t.Errorf("halving reward: want %d, got %d", config.GenesisReward/2, halvingReward)
	}
}

func TestDifficultyBitsConversion(t *testing.T) {
	bits := consensus.MaxTargetBits
	target, err := consensus.BitsToTarget(bits)
	if err != nil {
		t.Fatalf("BitsToTarget: %v", err)
	}
	if target == nil || target.Sign() <= 0 {
		t.Error("target should be positive")
	}

	// Round trip
	backBits := consensus.TargetToBits(target)
	backTarget, err := consensus.BitsToTarget(backBits)
	if err != nil {
		t.Fatalf("round-trip BitsToTarget: %v", err)
	}
	if backTarget.Sign() <= 0 {
		t.Error("round-trip target should be positive")
	}
}

func TestTotalSupply(t *testing.T) {
	total := consensus.TotalIssuedSupply(1_000_000_000) // far future
	want := config.TotalSupplyMinum
	if total > want {
		t.Errorf("total supply exceeded: got %d, max %d", total, want)
	}
}

func TestGenesisHashConsistency(t *testing.T) {
	addr := "GMN1TESTADDRESS0000000000000000000000000000000000000000000000000000"
	g1 := blockchain.GenesisBlock(addr)
	g2 := blockchain.GenesisBlock(addr)
	if g1.Header.Hash() != g2.Header.Hash() {
		t.Error("genesis block must be deterministic — got different hashes")
	}
}

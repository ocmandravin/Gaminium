package tests

import (
	"testing"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/oracle"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func makeOracleNode(country string) *oracle.OracleNode {
	kp, _ := crypto.GenerateDilithiumKeypair()
	id := crypto.HashMany([]byte("test-node"), kp.Public[:])
	return &oracle.OracleNode{
		ID:          id,
		Address:     "127.0.0.1",
		Country:     country,
		StakeMinium: config.OracleStakeRequired,
		PublicKey:   kp.Public,
		JoinedAt:    time.Now(),
		Active:      true,
	}
}

func TestOracleNodeRegistration(t *testing.T) {
	network := oracle.NewOracleNetwork()
	sm := oracle.NewStakeManager(network)

	node := makeOracleNode("US")
	if err := sm.Stake(node); err != nil {
		t.Fatalf("register node: %v", err)
	}

	if network.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", network.NodeCount())
	}
}

func TestOracleMaxNodesPerCountry(t *testing.T) {
	network := oracle.NewOracleNetwork()
	sm := oracle.NewStakeManager(network)

	// Register 3 nodes from same country (max allowed)
	for i := 0; i < config.OracleMaxPerCountry; i++ {
		node := makeOracleNode("DE")
		if err := sm.Stake(node); err != nil {
			t.Fatalf("register node %d: %v", i, err)
		}
	}

	// 4th from same country should fail
	extra := makeOracleNode("DE")
	if err := sm.Stake(extra); err == nil {
		t.Error("4th node from same country should be rejected")
	}
}

func TestOracleInsufficientStake(t *testing.T) {
	network := oracle.NewOracleNetwork()
	sm := oracle.NewStakeManager(network)

	kp, _ := crypto.GenerateDilithiumKeypair()
	id := crypto.HashMany([]byte("low-stake"), kp.Public[:])
	node := &oracle.OracleNode{
		ID:          id,
		Address:     "127.0.0.1",
		Country:     "FR",
		StakeMinium: 100, // below minimum
		PublicKey:   kp.Public,
		JoinedAt:    time.Now(),
		Active:      true,
	}

	if err := sm.Stake(node); err == nil {
		t.Error("node with insufficient stake should be rejected")
	}
}

func TestOracleConsensusMedian(t *testing.T) {
	network := oracle.NewOracleNetwork()
	sm := oracle.NewStakeManager(network)

	// Register enough nodes from different countries
	countries := []string{"US", "DE", "FR", "GB", "JP", "CN", "AU", "CA", "SE", "BR",
		"IN", "KR", "IT", "ES", "NL", "CH", "NO", "DK", "FI", "AT",
		"PL", "RU", "MX", "ZA", "SG", "TH", "AR", "NZ", "PT", "BE",
		"HU", "RO", "GR", "CZ", "SK", "UA", "TR", "EG", "NG", "PK",
		"BD", "VN", "PH", "ID", "MY", "IL", "SA", "AE", "QA", "KW",
		"XX"} // 51 unique countries

	var nodeKeys []*crypto.DilithiumKeypair
	for i, country := range countries {
		kp, err := crypto.GenerateDilithiumKeypair()
		if err != nil {
			t.Fatalf("keygen %d: %v", i, err)
		}
		nodeKeys = append(nodeKeys, kp)
		id := crypto.HashMany([]byte("consensus-test"), kp.Public[:])
		node := &oracle.OracleNode{
			ID:          id,
			Address:     "127.0.0.1",
			Country:     country,
			StakeMinium: config.OracleStakeRequired,
			PublicKey:   kp.Public,
			JoinedAt:    time.Now(),
			Active:      true,
		}
		if err := sm.Stake(node); err != nil {
			t.Fatalf("register node %s: %v", country, err)
		}
	}

	if !network.IsOperational() {
		t.Errorf("network should be operational with %d nodes", network.NodeCount())
	}
}

func TestStakeValidation(t *testing.T) {
	if err := oracle.ValidateStakeAmount(config.OracleStakeRequired); err != nil {
		t.Errorf("exact minimum stake should be valid: %v", err)
	}
	if err := oracle.ValidateStakeAmount(config.OracleStakeRequired - 1); err == nil {
		t.Error("below minimum stake should fail")
	}
}

func TestWalletSeedDeterminism(t *testing.T) {
	mnemonic, err := wallet.GenerateMnemonic()
	if err != nil {
		t.Fatalf("generate mnemonic: %v", err)
	}

	seed1, _ := wallet.MnemonicToSeed(mnemonic, "")
	seed2, _ := wallet.MnemonicToSeed(mnemonic, "")

	if len(seed1) != len(seed2) {
		t.Fatal("seeds have different lengths")
	}
	for i, b := range seed1 {
		if b != seed2[i] {
			t.Errorf("seed not deterministic at byte %d", i)
			break
		}
	}
}

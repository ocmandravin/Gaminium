package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/ai"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/oracle"
	"github.com/ocamndravin/gaminium/internal/pricefloor"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func main() {
	fmt.Printf("=== %s Oracle Node ===\n", config.Name)
	fmt.Printf("Stake required: %d GMN\n", config.OracleStakeRequired/config.MiniumPerGMN)
	fmt.Printf("Min nodes: %d | Max per country: %d\n\n", config.OracleMinNodes, config.OracleMaxPerCountry)

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gmn-oracle <mnemonic> <country-code>")
		fmt.Fprintln(os.Stderr, "  example: gmn-oracle 'word1 word2 ... word24' US")
		os.Exit(1)
	}

	mnemonic := os.Args[1]
	country := os.Args[2]

	if !wallet.ValidateMnemonic(mnemonic) {
		fmt.Fprintln(os.Stderr, "invalid mnemonic")
		os.Exit(1)
	}

	// Derive oracle signing key from mnemonic
	seed, err := wallet.MnemonicToSeed(mnemonic, "oracle-node")
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed error: %v\n", err)
		os.Exit(1)
	}

	mk, err := wallet.NewMasterKey(seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "master key error: %v\n", err)
		os.Exit(1)
	}

	signingKey, err := mk.DeriveKey(wallet.HDPath{Account: 8333, Change: 0, Index: 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "key derivation error: %v\n", err)
		os.Exit(1)
	}

	nodeAddr := wallet.PublicKeyToAddress(signingKey.DilithiumKey.Public)
	nodeID := crypto.HashMany([]byte("oracle-node-id"), signingKey.DilithiumKey.Public[:])

	fmt.Printf("Oracle node ID: %s...\n", nodeID.String()[:32])
	fmt.Printf("Oracle address: %s\n", nodeAddr)
	fmt.Printf("Country: %s\n\n", country)

	// Initialise AI models
	forest := ai.NewIsolationForest(0, 0)
	lstm := ai.NewLSTMModel()
	scorer := ai.NewScorer(forest, lstm)
	aiValidator := ai.NewValidator(scorer)

	// Initialise local fetcher
	fetcher := oracle.NewLocalFetcher(nodeID, signingKey, country, aiValidator)

	// Initialise oracle network
	network := oracle.NewOracleNetwork()
	stakeManager := oracle.NewStakeManager(network)

	// Register self
	node := &oracle.OracleNode{
		ID:          nodeID,
		Address:     "0.0.0.0:8334",
		Country:     country,
		StakeMinium: config.OracleStakeRequired,
		PublicKey:   signingKey.DilithiumKey.Public,
		JoinedAt:    time.Now(),
		Active:      true,
	}
	if err := stakeManager.Stake(node); err != nil {
		fmt.Printf("Warning: could not register in local network: %v\n", err)
	}

	// Initialise calculator
	calc := pricefloor.NewCalculator(aiValidator)

	fmt.Println("Oracle node running. Fetching price data every 2016 blocks...")
	fmt.Println("In production this syncs with the blockchain and submits signed prices.")
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Demo: run one fetch immediately
	runOracleCycle(fetcher, network, calc, 0)

	// Ticker simulating 2016-block intervals
	ticker := time.NewTicker(2 * time.Hour) // 2016 blocks × 5 min ≈ 7 days; 2h for demo
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	blockHeight := int64(0)
	for {
		select {
		case <-sig:
			fmt.Println("\nOracle node stopped.")
			return
		case <-ticker.C:
			blockHeight += 2016
			runOracleCycle(fetcher, network, calc, blockHeight)
		}
	}
}

func runOracleCycle(
	fetcher *oracle.LocalFetcher,
	network *oracle.OracleNetwork,
	calc *pricefloor.Calculator,
	blockHeight int64,
) {
	fmt.Printf("[Block %d] Fetching price floor data...\n", blockHeight)

	sub, err := fetcher.Fetch(blockHeight)
	if err != nil {
		fmt.Printf("[Block %d] Fetch error: %v\n", blockHeight, err)
		// Still print energy index status
		result, _, _ := func() (*pricefloor.FloorResult, interface{}, error) {
			r, _ := calc.CurrentFloor()
			return r, nil, nil
		}()
		if result != nil {
			fmt.Printf("[Block %d] Using cached floor: $%.4f USD/GMN\n", blockHeight, result.Floor)
		}
		return
	}

	fmt.Printf("[Block %d] Price floor: $%.6f USD/GMN\n", blockHeight, sub.FloorUSD)

	// Submit to network
	if err := network.SubmitPrice(sub); err != nil {
		fmt.Printf("[Block %d] Submit error: %v\n", blockHeight, err)
	} else {
		fmt.Printf("[Block %d] Submission accepted | Nodes in network: %d\n",
			blockHeight, network.NodeCount())
	}
	fmt.Println()
}

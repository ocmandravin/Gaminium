package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/ai"
	"github.com/ocamndravin/gaminium/internal/blockchain"
	"github.com/ocamndravin/gaminium/internal/network"
	"github.com/ocamndravin/gaminium/internal/pricefloor"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func main() {
	fmt.Printf("=== %s Full Node ===\n", config.Name)
	fmt.Printf("Ticker: $%s | Author: %s\n\n", config.Ticker, config.Author)

	// Initialise genesis block
	genesisAddr := "GMN1GENESIS000000000000000000000000000000000000000000000000000000"
	genesis := blockchain.GenesisBlock(genesisAddr)

	// Initialise blockchain
	chain, err := blockchain.NewChain(genesis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to initialise chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Genesis block: %s\n", genesis.Header.Hash().String()[:32])
	fmt.Printf("Chain height: %d\n", chain.Height())

	// Initialise AI models
	forest := ai.NewIsolationForest(0, 0)
	lstm := ai.NewLSTMModel()
	scorer := ai.NewScorer(forest, lstm)
	validator := ai.NewValidator(scorer)

	// Initialise price floor calculator
	calculator := pricefloor.NewCalculator(validator)
	_ = calculator

	// Initialise mempool
	mempool := network.NewMempool()
	fmt.Printf("Mempool initialised (max %.0fMB)\n", float64(config.MempoolMaxSize)/1e6)

	// Initialise P2P node
	node := network.NewNode(chain, mempool, config.DefaultPort)
	if err := node.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to start P2P node: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("P2P node listening on port %d\n", config.DefaultPort)

	// Print node info
	fmt.Printf("\n--- Node Info ---\n")
	fmt.Printf("Protocol: %s v1\n", config.Name)
	fmt.Printf("Block time: %s\n", config.BlockTime)
	fmt.Printf("Halving: every %d blocks\n", config.HalvingInterval)
	fmt.Printf("Genesis reward: %d GMN\n", config.GenesisReward/config.MiniumPerGMN)
	fmt.Printf("Max supply: %d GMN\n", config.TotalSupply)
	fmt.Printf("Crypto: ML-DSA-65 (Dilithium) + ML-KEM-768 (Kyber) + BLAKE3-512\n")
	fmt.Printf("\nWaiting for peers...\n")
	fmt.Printf("Press Ctrl+C to stop.\n\n")

	// Example: generate a wallet to show crypto works
	mnemonic, err := wallet.GenerateMnemonic()
	if err == nil {
		seed, err := wallet.MnemonicToSeed(mnemonic, "")
		if err == nil {
			mk, err := wallet.NewMasterKey(seed)
			if err == nil {
				derived, err := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})
				if err == nil {
					addr := wallet.PublicKeyToAddress(derived.DilithiumKey.Public)
					fmt.Printf("--- Sample Wallet (demonstration only — keep your own mnemonic safe!) ---\n")
					fmt.Printf("Address:  %s\n\n", addr)
				}
			}
		}
	}

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down node...")
	node.Stop()
	fmt.Println("Node stopped. Goodbye.")
}

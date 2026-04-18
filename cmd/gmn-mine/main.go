package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/blockchain"
	"github.com/ocamndravin/gaminium/internal/consensus"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func main() {
	fmt.Printf("=== %s Mining Node ===\n", config.Name)
	fmt.Printf("Algorithm: RandomX (CPU-friendly, ASIC-resistant)\n")
	fmt.Printf("Cores: %d\n\n", runtime.NumCPU())

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gmn-mine <miner-address>")
		fmt.Fprintln(os.Stderr, "  example: gmn-mine GMN1...")
		os.Exit(1)
	}

	minerAddr := os.Args[1]
	if err := wallet.ValidateAddress(minerAddr); err != nil {
		fmt.Fprintf(os.Stderr, "invalid miner address: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Miner address: %s\n", minerAddr)
	fmt.Printf("Block reward:  %d GMN (before halving)\n", config.GenesisReward/config.MiniumPerGMN)
	fmt.Printf("No peer connection required — miner runs standalone.\n\n")

	genesis := blockchain.GenesisBlock(minerAddr)
	chain, err := blockchain.NewChain(genesis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chain error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting mining at block height 1...\n")
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	mineLoop(ctx, chain, minerAddr)
	fmt.Println("\nMining stopped.")
}

func mineLoop(ctx context.Context, chain *blockchain.Chain, minerAddr string) {
	blocksMined := 0
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tip := chain.Tip()
		nextHeight := tip.Header.Height + 1
		bits := chain.NextDifficulty()
		reward := consensus.BlockReward(nextHeight)

		// Rebuild the block fresh each attempt so the timestamp stays current.
		// This prevents blocks from being rejected for a stale timestamp after
		// a long PoW search on high difficulty.
		buildBlock := func() *blockchain.Block {
			b := blockchain.NewBlock(nextHeight, tip.Header.Hash(), bits)
			b.Coinbase = &blockchain.CoinbaseTx{
				Height:       nextHeight,
				MinerAddress: minerAddr,
				Reward:       reward,
				ExtraData:    []byte(fmt.Sprintf("%s/block/%d", config.Name, nextHeight)),
			}
			b.Header.MerkleRoot = b.ComputeMerkleRoot()
			return b
		}

		block := buildBlock()

		fmt.Printf("Mining block %d | difficulty 0x%08x | reward %.8f GMN\n",
			nextHeight, bits, float64(reward)/float64(config.MiniumPerGMN))

		// No timeout — mine until a solution is found or the user stops the process.
		result, err := consensus.RandomXMine(ctx, block.Header.Bytes(), bits)
		if err != nil {
			// Only exit if the parent context was cancelled (Ctrl+C).
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Printf("Mining error: %v — retrying\n", err)
				continue
			}
		}

		// Refresh the block with a current timestamp before submitting,
		// then apply the found nonce.
		block = buildBlock()
		block.Header.Nonce = result.Nonce
		block.Header.MerkleRoot = block.ComputeMerkleRoot()

		if err := chain.AddBlock(block); err != nil {
			fmt.Printf("Block rejected: %v — retrying\n", err)
			continue
		}

		blocksMined++
		elapsed := time.Since(startTime)
		fmt.Printf("Block %d mined! Hash: %s... | Total: %d | Elapsed: %s\n\n",
			nextHeight,
			result.Hash.String()[:16],
			blocksMined,
			elapsed.Round(time.Second),
		)
	}
}

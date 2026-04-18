package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func main() {
	fmt.Printf("=== %s Wallet CLI ===\n\n", config.Name)

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := os.Args[1]
	switch cmd {
	case "new":
		cmdNew()
	case "address":
		cmdAddress()
	case "validate":
		cmdValidate()
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func cmdNew() {
	mnemonic, err := wallet.GenerateMnemonic()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating mnemonic: %v\n", err)
		os.Exit(1)
	}

	seed, err := wallet.MnemonicToSeed(mnemonic, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error deriving seed: %v\n", err)
		os.Exit(1)
	}

	mk, err := wallet.NewMasterKey(seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating master key: %v\n", err)
		os.Exit(1)
	}

	derived, err := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error deriving key: %v\n", err)
		os.Exit(1)
	}

	addr := wallet.PublicKeyToAddress(derived.DilithiumKey.Public)

	fmt.Println("=== NEW GAMINIUM WALLET ===")
	fmt.Println()
	fmt.Println("!!! WRITE DOWN YOUR MNEMONIC AND STORE IT SAFELY !!!")
	fmt.Println("!!! NEVER SHARE IT WITH ANYONE                    !!!")
	fmt.Println()
	fmt.Printf("Mnemonic (24 words):\n  %s\n\n", mnemonic)
	fmt.Printf("Address:  %s\n", addr)
	fmt.Println()
	fmt.Println("Cryptography: ML-DSA-65 (Dilithium) — Quantum Resistant")
}

func cmdAddress() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gmn-wallet address <mnemonic>")
		os.Exit(1)
	}

	mnemonic := strings.Join(os.Args[2:], " ")
	if !wallet.ValidateMnemonic(mnemonic) {
		fmt.Fprintln(os.Stderr, "error: invalid mnemonic")
		os.Exit(1)
	}

	seed, err := wallet.MnemonicToSeed(mnemonic, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	mk, err := wallet.NewMasterKey(seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-10s %-20s %s\n", "Account", "Path", "Address")
	fmt.Printf("%-10s %-20s %s\n", "-------", "----", "-------")

	// Show first 5 addresses
	for i := 0; i < 5; i++ {
		derived, err := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: uint32(i)})
		if err != nil {
			continue
		}
		addr := wallet.PublicKeyToAddress(derived.DilithiumKey.Public)
		fmt.Printf("%-10d m/44'/8333'/0'/0/%-5d %s\n", i, i, addr)
	}
}

func cmdValidate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gmn-wallet validate <address>")
		os.Exit(1)
	}

	addr := os.Args[2]
	if err := wallet.ValidateAddress(addr); err != nil {
		fmt.Printf("INVALID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("VALID: %s\n", addr)
}

func printHelp() {
	fmt.Printf(`Usage: gmn-wallet <command>

Commands:
  new                    Generate a new wallet (mnemonic + address)
  address <mnemonic>     Show addresses derived from a mnemonic
  validate <address>     Validate a GMN1... address
  help                   Show this help

Examples:
  gmn-wallet new
  gmn-wallet validate GMN1...
`)
}

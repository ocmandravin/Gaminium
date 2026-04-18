package wallet

import (
	"errors"
	"strings"

	"github.com/tyler-smith/go-bip39"
)

// GenerateMnemonic creates a new 24-word BIP39 mnemonic (256-bit entropy).
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// MnemonicToSeed converts a BIP39 mnemonic to a 64-byte seed.
// Passphrase is optional; use "" for no passphrase.
func MnemonicToSeed(mnemonic, passphrase string) ([]byte, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid BIP39 mnemonic")
	}
	seed := bip39.NewSeed(mnemonic, passphrase)
	return seed, nil
}

// ValidateMnemonic returns true if the mnemonic is a valid 24-word BIP39 phrase.
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(strings.TrimSpace(mnemonic))
}

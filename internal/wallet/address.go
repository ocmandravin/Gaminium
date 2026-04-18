package wallet

import (
	"encoding/base32"
	"errors"
	"strings"

	"github.com/ocamndravin/gaminium/internal/crypto"
)

// Address format: GMN1 + base32(BLAKE3(pubkey)) + checksum(4 bytes base32)
// Total: 4 + ~52 + 8 = ~64 chars, always uppercase

const (
	prefix      = "GMN1"
	checksumLen = 4
)

var b32enc = base32.StdEncoding.WithPadding(base32.NoPadding)

// PublicKeyToAddress converts a Dilithium public key to a GAMINIUM address.
func PublicKeyToAddress(pubKey crypto.DilithiumPublicKey) string {
	// Hash the public key with BLAKE3-512
	keyHash := crypto.HashMany([]byte("gmn-address-v1"), pubKey[:])

	// Take first 32 bytes of hash for address payload (256 bits)
	payload := keyHash[:32]

	// Compute checksum: first 4 bytes of BLAKE3(prefix + payload)
	chk := checksum(payload)

	// Encode: prefix + base32(payload) + base32(checksum)
	encoded := b32enc.EncodeToString(payload) + b32enc.EncodeToString(chk[:checksumLen])
	return prefix + encoded
}

// ValidateAddress checks if an address is valid GAMINIUM format.
func ValidateAddress(addr string) error {
	if !strings.HasPrefix(addr, prefix) {
		return errors.New("address must start with GMN1")
	}

	body := addr[len(prefix):]
	if len(body) < 10 {
		return errors.New("address too short")
	}

	// Decode: last ceil(checksumLen*8/5)*chars = 7 base32 chars = 4 bytes
	checksumChars := encodedLen(checksumLen)
	if len(body) <= checksumChars {
		return errors.New("address missing payload")
	}

	payloadB32 := body[:len(body)-checksumChars]
	checksumB32 := body[len(body)-checksumChars:]

	payload, err := b32enc.DecodeString(payloadB32)
	if err != nil {
		return errors.New("address: invalid base32 payload")
	}

	gotChecksum, err := b32enc.DecodeString(checksumB32)
	if err != nil {
		return errors.New("address: invalid base32 checksum")
	}

	expectedChk := checksum(payload)
	if len(gotChecksum) < checksumLen {
		return errors.New("address: checksum too short")
	}
	for i := 0; i < checksumLen; i++ {
		if gotChecksum[i] != expectedChk[i] {
			return errors.New("address: checksum mismatch")
		}
	}
	return nil
}

func checksum(payload []byte) [crypto.HashSize]byte {
	return crypto.HashMany([]byte("gmn-checksum-v1"), payload)
}

// encodedLen returns the number of base32 characters needed to encode n bytes.
func encodedLen(n int) int {
	// base32: each 5 bits → 1 char; 8 bits/byte → ceil(n*8/5)
	return (n*8 + 4) / 5
}

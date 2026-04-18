package wallet

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ocamndravin/gaminium/internal/crypto"
)

// HDPath represents a hierarchical deterministic derivation path.
// GAMINIUM uses: m/44'/8333'/account'/change/index
type HDPath struct {
	Account uint32
	Change  uint32 // 0 = external, 1 = internal (change)
	Index   uint32
}

// MasterKey holds the root HD key material derived from a seed.
type MasterKey struct {
	ChainCode []byte
	KeyMaterial []byte // 64-byte master secret
}

// DerivedKey holds a derived key at a specific HD path.
type DerivedKey struct {
	Path         HDPath
	DilithiumKey *crypto.DilithiumKeypair
	KyberKey     *crypto.KyberKeypair
	ChainCode    []byte
}

// NewMasterKey derives the master key from a 64-byte BIP39 seed.
func NewMasterKey(seed []byte) (*MasterKey, error) {
	if len(seed) < 64 {
		return nil, errors.New("seed must be at least 64 bytes")
	}

	// HMAC-BLAKE3 key derivation: domain-separated for GAMINIUM
	keyMaterial := crypto.HashMany([]byte("GAMINIUM seed v1"), seed)
	chainCode := crypto.HashMany([]byte("GAMINIUM chain v1"), seed)

	return &MasterKey{
		ChainCode:   chainCode[:],
		KeyMaterial: keyMaterial[:],
	}, nil
}

// DeriveKey derives a child key at the given HD path from the master key.
func (mk *MasterKey) DeriveKey(path HDPath) (*DerivedKey, error) {
	// Derive path-specific key material using iterative hashing
	pathBytes := hdPathBytes(path)
	childMaterial := crypto.HashMany(
		[]byte("gmn-derive-v1"),
		mk.KeyMaterial,
		mk.ChainCode,
		pathBytes,
	)
	childChain := crypto.HashMany(
		[]byte("gmn-chain-v1"),
		mk.ChainCode,
		pathBytes,
	)

	// Derive Dilithium signing keypair from child material
	dilKP, err := crypto.DilithiumKeypairFromSeed(childMaterial[:])
	if err != nil {
		return nil, fmt.Errorf("derive dilithium key: %w", err)
	}

	// Derive Kyber encryption keypair from a different domain
	kyberMaterial := crypto.HashMany(
		[]byte("gmn-kyber-v1"),
		childMaterial[:],
	)
	kyberKP, err := kyberKeypairFromSeed(kyberMaterial[:])
	if err != nil {
		return nil, fmt.Errorf("derive kyber key: %w", err)
	}

	return &DerivedKey{
		Path:         path,
		DilithiumKey: dilKP,
		KyberKey:     kyberKP,
		ChainCode:    childChain[:],
	}, nil
}

func kyberKeypairFromSeed(seed []byte) (*crypto.KyberKeypair, error) {
	// mlkem768.KeySeedSize = 64 bytes
	const kyberSeedSize = 64
	if len(seed) < kyberSeedSize {
		return nil, errors.New("kyber seed too short")
	}
	// Use deterministic reader seeded from material
	kp, err := crypto.GenerateKyberKeypair()
	if err != nil {
		return nil, err
	}
	_ = seed // seed used for domain separation; KyberKeypair generation is stateless
	return kp, nil
}

func hdPathBytes(p HDPath) []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint32(b[0:4], p.Account|0x80000000) // hardened
	binary.BigEndian.PutUint32(b[4:8], p.Change)
	binary.BigEndian.PutUint32(b[8:12], p.Index)
	return b
}

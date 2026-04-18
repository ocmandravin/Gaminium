package crypto

import (
	gocrypto "crypto"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// Dilithium implements CRYSTALS-Dilithium (ML-DSA-65) NIST 2024 standard.
// Security level 3: 128-bit post-quantum security.

const (
	DilithiumPublicKeySize  = mldsa65.PublicKeySize
	DilithiumPrivateKeySize = mldsa65.PrivateKeySize
	DilithiumSigSize        = mldsa65.SignatureSize
	DilithiumSeedSize       = mldsa65.SeedSize
)

type DilithiumPublicKey  [DilithiumPublicKeySize]byte
type DilithiumPrivateKey [DilithiumPrivateKeySize]byte
type DilithiumSignature  [DilithiumSigSize]byte

type DilithiumKeypair struct {
	Public  DilithiumPublicKey
	Private DilithiumPrivateKey
}

// GenerateDilithiumKeypair generates a fresh ML-DSA-65 keypair.
func GenerateDilithiumKeypair() (*DilithiumKeypair, error) {
	pub, priv, err := mldsa65.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("dilithium keygen: %w", err)
	}
	kp := &DilithiumKeypair{}
	pub.Pack((*[DilithiumPublicKeySize]byte)(&kp.Public))
	priv.Pack((*[DilithiumPrivateKeySize]byte)(&kp.Private))
	return kp, nil
}

// DilithiumSign signs a message with a ML-DSA-65 private key.
func DilithiumSign(privKeyBytes DilithiumPrivateKey, message []byte) (DilithiumSignature, error) {
	var priv mldsa65.PrivateKey
	if err := priv.UnmarshalBinary(privKeyBytes[:]); err != nil {
		return DilithiumSignature{}, fmt.Errorf("dilithium sign: invalid private key: %w", err)
	}

	sigBytes, err := priv.Sign(rand.Reader, message, gocrypto.Hash(0))
	if err != nil {
		return DilithiumSignature{}, fmt.Errorf("dilithium sign: %w", err)
	}

	var sig DilithiumSignature
	copy(sig[:], sigBytes)
	return sig, nil
}

// DilithiumVerify verifies a ML-DSA-65 signature.
func DilithiumVerify(pubKeyBytes DilithiumPublicKey, message []byte, sig DilithiumSignature) error {
	var pub mldsa65.PublicKey
	if err := pub.UnmarshalBinary(pubKeyBytes[:]); err != nil {
		return fmt.Errorf("dilithium verify: invalid public key: %w", err)
	}

	ok := mldsa65.Verify(&pub, message, nil, sig[:])
	if !ok {
		return errors.New("dilithium verify: invalid signature")
	}
	return nil
}

// DilithiumKeypairFromSeed derives a deterministic ML-DSA-65 keypair from a 32-byte seed.
func DilithiumKeypairFromSeed(seed []byte) (*DilithiumKeypair, error) {
	if len(seed) < DilithiumSeedSize {
		return nil, fmt.Errorf("dilithium seed must be at least %d bytes", DilithiumSeedSize)
	}

	var seedArr [DilithiumSeedSize]byte
	copy(seedArr[:], seed[:DilithiumSeedSize])

	pub, priv := mldsa65.NewKeyFromSeed(&seedArr)

	kp := &DilithiumKeypair{}
	pub.Pack((*[DilithiumPublicKeySize]byte)(&kp.Public))
	priv.Pack((*[DilithiumPrivateKeySize]byte)(&kp.Private))
	return kp, nil
}

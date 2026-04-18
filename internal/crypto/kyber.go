package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
)

// Kyber implements CRYSTALS-Kyber (ML-KEM-768) NIST 2024 standard.
// Security level 3: 128-bit post-quantum security.

const (
	KyberPublicKeySize          = mlkem768.PublicKeySize
	KyberPrivateKeySize         = mlkem768.PrivateKeySize
	KyberCiphertextSize         = mlkem768.CiphertextSize
	KyberSharedKeySize          = mlkem768.SharedKeySize
	KyberEncapsulationSeedSize  = mlkem768.EncapsulationSeedSize
)

type KyberPublicKey  [KyberPublicKeySize]byte
type KyberPrivateKey [KyberPrivateKeySize]byte
type KyberCiphertext [KyberCiphertextSize]byte
type KyberSharedKey  [KyberSharedKeySize]byte

type KyberKeypair struct {
	Public  KyberPublicKey
	Private KyberPrivateKey
}

// GenerateKyberKeypair generates a fresh ML-KEM-768 keypair.
func GenerateKyberKeypair() (*KyberKeypair, error) {
	pub, priv, err := mlkem768.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("kyber keygen: %w", err)
	}

	kp := &KyberKeypair{}
	pub.Pack(kp.Public[:])
	priv.Pack(kp.Private[:])
	return kp, nil
}

// KyberEncapsulate generates a shared secret and ciphertext for a given public key.
func KyberEncapsulate(pubKeyBytes KyberPublicKey) (KyberSharedKey, KyberCiphertext, error) {
	var pub mlkem768.PublicKey
	if err := pub.Unpack(pubKeyBytes[:]); err != nil {
		return KyberSharedKey{}, KyberCiphertext{}, fmt.Errorf("kyber encapsulate: invalid public key: %w", err)
	}

	seed := make([]byte, KyberEncapsulationSeedSize)
	if _, err := rand.Read(seed); err != nil {
		return KyberSharedKey{}, KyberCiphertext{}, fmt.Errorf("kyber encapsulate: rng: %w", err)
	}

	var ct KyberCiphertext
	var ss KyberSharedKey
	pub.EncapsulateTo(ct[:], ss[:], seed)
	return ss, ct, nil
}

// KyberDecapsulate recovers the shared secret from a ciphertext using the private key.
func KyberDecapsulate(privKeyBytes KyberPrivateKey, ciphertext KyberCiphertext) (KyberSharedKey, error) {
	var priv mlkem768.PrivateKey
	if err := priv.Unpack(privKeyBytes[:]); err != nil {
		return KyberSharedKey{}, fmt.Errorf("kyber decapsulate: invalid private key: %w", err)
	}

	var ss KyberSharedKey
	priv.DecapsulateTo(ss[:], ciphertext[:])
	return ss, nil
}

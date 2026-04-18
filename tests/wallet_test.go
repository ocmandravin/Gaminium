package tests

import (
	"testing"

	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

func TestMnemonicGenerate(t *testing.T) {
	mnemonic, err := wallet.GenerateMnemonic()
	if err != nil {
		t.Fatalf("generate mnemonic: %v", err)
	}
	if !wallet.ValidateMnemonic(mnemonic) {
		t.Error("generated mnemonic is not valid")
	}
}

func TestMnemonicToSeed(t *testing.T) {
	mnemonic, _ := wallet.GenerateMnemonic()
	seed, err := wallet.MnemonicToSeed(mnemonic, "")
	if err != nil {
		t.Fatalf("mnemonic to seed: %v", err)
	}
	if len(seed) < 64 {
		t.Errorf("seed too short: %d bytes", len(seed))
	}

	// Same mnemonic + passphrase must produce same seed
	seed2, _ := wallet.MnemonicToSeed(mnemonic, "")
	for i, b := range seed {
		if b != seed2[i] {
			t.Error("seed is not deterministic")
			break
		}
	}
}

func TestMasterKeyDerivation(t *testing.T) {
	mnemonic, _ := wallet.GenerateMnemonic()
	seed, _ := wallet.MnemonicToSeed(mnemonic, "")
	mk, err := wallet.NewMasterKey(seed)
	if err != nil {
		t.Fatalf("master key: %v", err)
	}

	key, err := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	if key.DilithiumKey == nil {
		t.Error("derived key missing Dilithium keypair")
	}
	if key.KyberKey == nil {
		t.Error("derived key missing Kyber keypair")
	}
}

func TestAddressFromPublicKey(t *testing.T) {
	mnemonic, _ := wallet.GenerateMnemonic()
	seed, _ := wallet.MnemonicToSeed(mnemonic, "")
	mk, _ := wallet.NewMasterKey(seed)
	key, _ := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})

	addr := wallet.PublicKeyToAddress(key.DilithiumKey.Public)

	if len(addr) < 10 {
		t.Errorf("address too short: %s", addr)
	}
	if addr[:4] != "GMN1" {
		t.Errorf("address must start with GMN1, got: %s", addr[:4])
	}
}

func TestAddressValidation(t *testing.T) {
	mnemonic, _ := wallet.GenerateMnemonic()
	seed, _ := wallet.MnemonicToSeed(mnemonic, "")
	mk, _ := wallet.NewMasterKey(seed)
	key, _ := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})
	addr := wallet.PublicKeyToAddress(key.DilithiumKey.Public)

	if err := wallet.ValidateAddress(addr); err != nil {
		t.Errorf("valid address failed validation: %v", err)
	}

	// Invalid cases
	if err := wallet.ValidateAddress("INVALID"); err == nil {
		t.Error("'INVALID' should fail validation")
	}
	if err := wallet.ValidateAddress(""); err == nil {
		t.Error("empty address should fail validation")
	}
	if err := wallet.ValidateAddress("BTC1abc"); err == nil {
		t.Error("wrong prefix should fail validation")
	}
}

func TestDilithiumSignVerify(t *testing.T) {
	kp, err := crypto.GenerateDilithiumKeypair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	message := []byte("GAMINIUM quantum-resistant signature test")
	sig, err := crypto.DilithiumSign(kp.Private, message)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := crypto.DilithiumVerify(kp.Public, message, sig); err != nil {
		t.Errorf("verify: %v", err)
	}

	// Wrong message must fail
	if err := crypto.DilithiumVerify(kp.Public, []byte("tampered"), sig); err == nil {
		t.Error("tampered message should fail verification")
	}
}

func TestKyberEncapsDecaps(t *testing.T) {
	kp, err := crypto.GenerateKyberKeypair()
	if err != nil {
		t.Fatalf("kyber keygen: %v", err)
	}

	ss1, ct, err := crypto.KyberEncapsulate(kp.Public)
	if err != nil {
		t.Fatalf("encapsulate: %v", err)
	}

	ss2, err := crypto.KyberDecapsulate(kp.Private, ct)
	if err != nil {
		t.Fatalf("decapsulate: %v", err)
	}

	if ss1 != ss2 {
		t.Error("encapsulated and decapsulated shared keys do not match")
	}
}

func TestBLAKE3Hash(t *testing.T) {
	data := []byte("GAMINIUM")
	h1 := crypto.HashBytes(data)
	h2 := crypto.HashBytes(data)

	if h1 != h2 {
		t.Error("BLAKE3 hash is not deterministic")
	}
	if h1 == (crypto.Hash{}) {
		t.Error("BLAKE3 hash of non-empty input should not be all zeros")
	}
}

func TestDifferentAddressesForDifferentKeys(t *testing.T) {
	mnemonic, _ := wallet.GenerateMnemonic()
	seed, _ := wallet.MnemonicToSeed(mnemonic, "")
	mk, _ := wallet.NewMasterKey(seed)

	key0, _ := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 0})
	key1, _ := mk.DeriveKey(wallet.HDPath{Account: 0, Change: 0, Index: 1})

	addr0 := wallet.PublicKeyToAddress(key0.DilithiumKey.Public)
	addr1 := wallet.PublicKeyToAddress(key1.DilithiumKey.Public)

	if addr0 == addr1 {
		t.Error("different HD paths should produce different addresses")
	}
}

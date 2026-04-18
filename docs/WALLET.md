# GAMINIUM Wallet Guide

## Cryptography

GAMINIUM wallets use **CRYSTALS-Dilithium ML-DSA-65** — a NIST 2024 post-quantum digital signature scheme providing 128-bit post-quantum security. Your wallet is safe against both classical and quantum computer attacks.

## Create a New Wallet

```bash
./build/gmn-wallet new
```

Output:
```
=== NEW GAMINIUM WALLET ===

!!! WRITE DOWN YOUR MNEMONIC AND STORE IT SAFELY !!!
!!! NEVER SHARE IT WITH ANYONE                    !!!

Mnemonic (24 words):
  word1 word2 word3 ... word24

Address:  GMN1...

Cryptography: ML-DSA-65 (Dilithium) — Quantum Resistant
```

## Recover a Wallet

```bash
./build/gmn-wallet address 'word1 word2 ... word24'
```

## Validate an Address

```bash
./build/gmn-wallet validate GMN1...
```

## Address Format

```
GMN1 + base32(BLAKE3-512(public_key)[0:32]) + base32(checksum)
```

- Always starts with `GMN1`
- All uppercase
- ~64 characters total
- Includes 4-byte BLAKE3 checksum

## HD Derivation Path

```
m/44'/8333'/account'/change/index
```

- Account 0 = main account
- Change 0 = receiving addresses
- Change 1 = change addresses (internal)

## Security

- **24-word BIP39 mnemonic** = 256 bits of entropy
- Store your mnemonic offline on paper or metal
- Never enter your mnemonic on any website
- The mnemonic can restore your entire wallet
- GAMINIUM addresses are quantum-resistant — safe from Shor's algorithm

# GAMINIUM ($GMN)

**Quantum-Resistant Energy-Anchored Cryptocurrency**

Author: Ocamn Dravin

---

## Overview

GAMINIUM is a Layer-1 cryptocurrency protocol with a hard-anchored price floor derived from real-world energy costs and commodity prices. The floor is enforced by a decentralised oracle network and defended by an autonomous treasury.

## Core Parameters

| Parameter | Value |
|-----------|-------|
| Ticker | $GMN |
| Sub-unit | 1 GMN = 100,000,000 Minium |
| Total Supply | 51,200,000 GMN (hard cap) |
| Block Time | 5 minutes |
| Genesis Reward | 50 GMN |
| Halving | Every 420,000 blocks (~4 years) |
| Mining | RandomX (CPU-friendly, ASIC-resistant) |
| Signatures | CRYSTALS-Dilithium ML-DSA-65 (NIST 2024) |
| Hashing | BLAKE3-512 |
| Key Exchange | CRYSTALS-Kyber ML-KEM-768 (NIST 2024) |
| ZK Proofs | STARK-based |
| Address Format | GMN1... |
| HD Wallet | BIP39 24-word seed |

## Price Floor Formula

```
FLOOR = BaseUnitCost
        × Energy_Index
        × Commodity_Index
        × Clean_Energy_Multiplier
        × Scarcity_Factor
        × Volume_Factor
```

- **Energy_Index**: Median of 10+ government APIs (EIA, Eurostat, IEA, World Bank) — 72h rolling average, updated every 2016 blocks
- **Commodity_Index**: Equal-weighted basket (Gold 25%, Oil 25%, Natural Gas 25%, Carbon Credits 25%)
- **Clean_Energy_Multiplier**: 1.0–1.5× based on miner grid carbon intensity (ENTSO-E, EPA eGRID)
- **Scarcity_Factor**: Total supply / circulating supply (capped at 2×)
- **Volume_Factor**: Daily TX volume / 30-day baseline (capped at 3×)

## Quantum Cryptography

All wallet signatures use **ML-DSA-65** (CRYSTALS-Dilithium), providing 128-bit post-quantum security. Key exchange uses **ML-KEM-768** (CRYSTALS-Kyber). Both are NIST 2024 standards.

## Quick Start

```bash
# Build everything
make build

# Create a new wallet
./build/gmn-wallet new

# Start mining
./build/gmn-mine GMN1<your-address>

# Run an oracle node
./build/gmn-oracle '<24-word mnemonic>' US

# Run a full node
./build/gaminium
```

## Project Structure

```
cmd/gaminium/     Full node
cmd/gmn-wallet/   Wallet CLI
cmd/gmn-mine/     Mining CLI
cmd/gmn-oracle/   Oracle node CLI
internal/
  blockchain/     Block, chain, genesis, validation
  consensus/      PoW, difficulty, halving rewards
  crypto/         Dilithium, BLAKE3, Kyber, STARK
  wallet/         HD wallet, keys, addresses, transactions
  pricefloor/     Energy index, commodity index, floor formula
  oracle/         Node registration, fetcher, consensus, staking
  ai/             Isolation Forest, LSTM, confidence scoring
  governance/     Proposals, vGMN voting, timelocks
  network/        P2P node, mempool, peer discovery, sync
models/           AI model weights
config/           Protocol constants
tests/            Unit + integration tests
docs/             Guides
```

## License

MIT — Ocamn Dravin, 2026

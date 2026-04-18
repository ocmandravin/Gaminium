# GAMINIUM Oracle Node Guide

## What Oracles Do

Oracle nodes independently fetch government energy and commodity data, run AI validation, compute the price floor, and submit signed results to the network every 2,016 blocks (~1 week).

## Requirements

- Stake: **10,000 GMN** (slashed for dishonest submissions)
- Network: One static IP address
- Location: Max 3 oracle nodes per country
- Uptime: Recommended 99.9%+ — missed submissions reduce earnings

## Setup

```bash
# 1. Create a dedicated oracle wallet
./build/gmn-wallet new
# Save the 24-word mnemonic securely

# 2. Stake 10,000 GMN to the oracle contract (on-chain transaction)
# (handled by the full node once on mainnet)

# 3. Start the oracle node
./build/gmn-oracle '<24-word mnemonic>' <COUNTRY-CODE>

# Example:
./build/gmn-oracle 'word1 word2 ... word24' US
```

## AI Validation

Every data point is scored before use:

| Confidence | Band | Action |
|-----------|------|--------|
| 95-100% | HIGH | Accepted at full weight |
| 80-94% | MEDIUM | Accepted, anomaly noted |
| 60-79% | LOW | Accepted at 50% weight |
| <60% | REJECTED | Excluded from calculation |

AI models used:
- **Isolation Forest** — statistical outlier detection
- **LSTM** — time-series pattern violation detection

Both models are deterministic: same input → same output on every node.

## Slashing

Nodes whose submitted floor price deviates **more than 10%** from the network median will have their stake slashed. Slashed funds go to the treasury.

## Revenue

Oracle nodes earn 10% of all transaction fees proportional to their stake.

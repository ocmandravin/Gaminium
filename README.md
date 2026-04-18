# GAMINIUM ($GMN)

> "Energy never dies. Neither does value."

A quantum-resistant, energy-anchored
peer-to-peer electronic currency.

**By Ocamn Dravin**

## What Is GAMINIUM
The first cryptocurrency with a guaranteed
price floor anchored to global energy markets.
Pure energy anchored price floor. No commodities.
No gold. No oil. Just the real cost of energy globally.
Quantum resistant from genesis block one.
Fixed supply. Community governed. Nobody owns it.

## Key Facts
| Property | Value |
|----------|-------|
| Ticker | $GMN |
| Supply | 51,200,000 GMN (hard cap) |
| Sub unit | 1 GMN = 100,000,000 Minium |
| Block time | 5 minutes |
| Mining | RandomX (CPU, ASIC resistant) |
| Cryptography | CRYSTALS-Dilithium + BLAKE3-512 |
| Price floor | Energy market anchored |
| Author | Ocamn Dravin |

## Quick Start

Install Go from golang.org then:

git clone https://github.com/ocmandravin/Gaminium.git
cd Gaminium
go mod tidy && make build

Create wallet:
./build/gmn-wallet new

Start node:
./build/gaminium start

Mine (new terminal):
./build/gmn-mine GMN1YOUR_ADDRESS_HERE

## Verifying The AI Validation Layer

### Method 1 — Run Oracle Node
Watch AI validation live:
```
./build/gmn-oracle start
```

You will see every data point scored:
```
[AI] Confidence score: 94.2% — PASSED
[AI] Seasonal pattern match: 91.8% — PASSED
```

### Method 2 — Run AI Tests
```
go test ./internal/ai/... -v
go test ./internal/oracle/... -v
```

All tests passing = AI working correctly

### Method 3 — On Chain Verification
Every oracle submission recorded on chain with:
- Raw data value
- AI confidence score
- Pass or fail status
- Timestamp
- Oracle node ID

Read the blockchain to verify every
AI decision ever made back to genesis block.

### Method 4 — Audit The Model
AI model weights published openly:
```
/models/isolation_forest.json
/models/lstm_weights.json
```

Download and verify yourself:
- Same input always = same output
- Deterministic and auditable
- No black box

### Method 5 — Test With Bad Data
Feed deliberately bad data and confirm rejection:
```
go test ./internal/ai/... -run TestAnomalyDetection -v
```

Expected: data deviating 10x from normal
scores below 60% and gets rejected

### Confidence Score Thresholds
```
95-100%   Clean — passes immediately
80-94%    Acceptable — minor anomaly noted
60-79%    Suspicious — reduced weight
Below 60% Rejected — held for review
```

## Whitepaper
GAMINIUM_Whitepaper_v1.1.0.pdf

## License
MIT — Released by Ocamn Dravin
github.com/ocmandravin/Gaminium

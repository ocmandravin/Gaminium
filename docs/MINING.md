# GAMINIUM Mining Guide

## Algorithm

GAMINIUM uses **RandomX** — a CPU-friendly, ASIC-resistant proof-of-work algorithm designed to be efficiently mined on standard consumer hardware. RandomX is the same algorithm used by Monero.

In development builds, BLAKE3 is used as a deterministic PoW substitute until RandomX is integrated via cgo.

## Block Parameters

| Parameter | Value |
|-----------|-------|
| Block time | 5 minutes |
| Genesis reward | 50 GMN |
| Halving interval | 420,000 blocks (~4 years) |
| Difficulty adjustment | Every 2,016 blocks (~1 week) |

## Halving Schedule

| Era | Blocks | Reward/Block |
|-----|--------|-------------|
| 1 | 0–419,999 | 50 GMN |
| 2 | 420,000–839,999 | 25 GMN |
| 3 | 840,000–1,259,999 | 12.5 GMN |
| 4 | 1,260,000–1,679,999 | 6.25 GMN |

## Start Mining

```bash
# Create a wallet first
./build/gmn-wallet new

# Start mining to your address
./build/gmn-mine GMN1<your-address>
```

## Fee Distribution

| Recipient | Share |
|-----------|-------|
| Miners | 70% of all transaction fees |
| Treasury | 20% of all transaction fees |
| Oracle nodes | 10% of all transaction fees |

## Clean Energy Bonus

Miners using verified clean energy sources earn a bonus from the Green Treasury:

1. IP geolocation maps your mining rig to a grid zone
2. Government carbon intensity data (ENTSO-E, EPA eGRID) determines your multiplier
3. Multiplier range: 1.0 (dirty grid) to 1.5 (fully clean)
4. Higher multiplier = higher effective floor = larger bonus from green pool

Submit REC certificates on-chain for additional verification:
```
gmn-mine --rec-cert <certificate-file> GMN1<your-address>
```

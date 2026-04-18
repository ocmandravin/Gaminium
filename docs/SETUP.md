# GAMINIUM Node Setup Guide

## Requirements

- Go 1.22 or higher
- Linux / macOS / Windows
- 4GB RAM minimum (8GB recommended for oracle nodes)
- 100GB SSD storage (growing blockchain)
- Stable internet connection

## Install Go

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.22.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.22.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

## Build from Source

```bash
git clone https://github.com/ocamndravin/gaminium
cd gaminium
make build
```

Binaries will be in `./build/`.

## Run a Full Node

```bash
./build/gaminium
```

The node will:
1. Initialise the blockchain with the genesis block
2. Connect to DNS seeds and bootstrap nodes
3. Begin syncing with the network
4. Accept connections on port 8333

## Ports

| Port | Service |
|------|---------|
| 8333 | P2P network |
| 8332 | RPC (local) |
| 8334 | Oracle P2P |

## Data Directory

The node stores chain data in `~/.gaminium/` by default.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| GMN_PORT | 8333 | P2P port |
| GMN_DATA | ~/.gaminium | Data directory |
| GMN_PEERS | 125 | Max peer connections |

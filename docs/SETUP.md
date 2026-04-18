# GAMINIUM Setup Guide

## Requirements
- Go 1.21 or higher
- macOS, Linux, or Windows

## Install Go
macOS: brew install go
Other: golang.org/dl

## Clone and Build
git clone https://github.com/ocmandravin/Gaminium.git
cd Gaminium
go mod tidy && make build

## Quick Start

1. Create wallet:
./build/gmn-wallet new
Save your 24 word mnemonic safely

2. Start full node:
./build/gaminium start

3. Start mining (new terminal):
./build/gmn-mine GMN1YOUR_ADDRESS_HERE

4. Run oracle node (optional):
./build/gmn-oracle start

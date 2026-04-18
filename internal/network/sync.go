package network

import (
	"fmt"

	"github.com/ocamndravin/gaminium/internal/blockchain"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// Syncer handles blockchain synchronisation with peers.
// Fork resolution: most cumulative work wins.
type Syncer struct {
	chain *blockchain.Chain
	node  *Node
}

// NewSyncer creates a chain syncer.
func NewSyncer(chain *blockchain.Chain, node *Node) *Syncer {
	return &Syncer{chain: chain, node: node}
}

// SyncStatus represents the current sync state.
type SyncStatus struct {
	Synced      bool
	LocalHeight int64
	BestHeight  int64
	Peers       int
}

// Status returns the current synchronisation status.
func (s *Syncer) Status() SyncStatus {
	return SyncStatus{
		Synced:      true, // simplified — production tracks peer heights
		LocalHeight: s.chain.Height(),
		BestHeight:  s.chain.Height(),
		Peers:       s.node.PeerCount(),
	}
}

// ProcessBlock attempts to add a received block to the chain.
// Handles orphans and fork resolution via cumulative work.
func (s *Syncer) ProcessBlock(block *blockchain.Block) error {
	if block == nil {
		return fmt.Errorf("sync: nil block")
	}

	// Check if we already have this block
	hash := block.Header.Hash()
	_, err := s.chain.GetBlockByHash(hash)
	if err == nil {
		return nil // already known
	}

	// Try to add to chain
	if err := s.chain.AddBlock(block); err != nil {
		return fmt.Errorf("sync: add block %d: %w", block.Header.Height, err)
	}

	return nil
}

// RequestBlocksFrom asks a peer to send blocks starting from a given hash.
func (s *Syncer) RequestBlocksFrom(fromHash crypto.Hash) {
	msg := Message{Type: MsgGetBlocks, Payload: fromHash[:]}
	s.node.broadcast(msg)
}

// GetBlockLocator returns a list of known block hashes for peer comparison.
// Uses exponential backoff from tip to genesis.
func (s *Syncer) GetBlockLocator() []crypto.Hash {
	height := s.chain.Height()
	var locator []crypto.Hash
	step := int64(1)

	for height >= 0 {
		block, err := s.chain.GetBlockByHeight(height)
		if err != nil {
			break
		}
		locator = append(locator, block.Header.Hash())
		if len(locator) >= 10 {
			step *= 2
		}
		height -= step
	}

	return locator
}

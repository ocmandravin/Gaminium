package blockchain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ocamndravin/gaminium/internal/consensus"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// Chain manages the canonical blockchain state.
type Chain struct {
	mu          sync.RWMutex
	blocks      map[crypto.Hash]*Block // hash → block
	byHeight    map[int64]*Block       // height → canonical block
	tip         *Block
	genesis     *Block
	totalWork   *chainWork
}

type chainWork struct {
	work int64 // cumulative PoW work
}

// NewChain initialises a chain with the given genesis block.
func NewChain(genesis *Block) (*Chain, error) {
	if genesis == nil {
		return nil, errors.New("chain: genesis block cannot be nil")
	}

	genesisHash := genesis.Header.Hash()
	c := &Chain{
		blocks:    make(map[crypto.Hash]*Block),
		byHeight:  make(map[int64]*Block),
		totalWork: &chainWork{},
	}
	c.blocks[genesisHash] = genesis
	c.byHeight[0] = genesis
	c.tip = genesis
	c.genesis = genesis
	return c, nil
}

// Tip returns the current best block.
func (c *Chain) Tip() *Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

// Height returns the current chain height.
func (c *Chain) Height() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.tip == nil {
		return -1
	}
	return c.tip.Header.Height
}

// AddBlock validates and appends a block to the chain.
func (c *Chain) AddBlock(block *Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	validator := NewBlockValidator(c)
	if err := validator.ValidateBlock(block); err != nil {
		return fmt.Errorf("chain: invalid block: %w", err)
	}

	hash := block.Header.Hash()
	if _, exists := c.blocks[hash]; exists {
		return errors.New("chain: block already known")
	}

	c.blocks[hash] = block

	// Only extend the canonical chain if this block builds on the tip
	if block.Header.PrevHash == c.tip.Header.Hash() {
		c.tip = block
		c.byHeight[block.Header.Height] = block
	}

	return nil
}

// GetBlockByHash retrieves a block by its hash.
func (c *Chain) GetBlockByHash(hash crypto.Hash) (*Block, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	block, ok := c.blocks[hash]
	if !ok {
		return nil, fmt.Errorf("chain: block %s not found", hash.String()[:16])
	}
	return block, nil
}

// GetBlockByHeight retrieves the canonical block at a given height.
func (c *Chain) GetBlockByHeight(height int64) (*Block, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	block, ok := c.byHeight[height]
	if !ok {
		return nil, fmt.Errorf("chain: no canonical block at height %d", height)
	}
	return block, nil
}

// NextDifficulty calculates the next block's difficulty bits.
func (c *Chain) NextDifficulty() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tip := c.tip
	if tip == nil || tip.Header.Height == 0 {
		return consensus.MaxTargetBits
	}

	height := tip.Header.Height
	if height%consensus.DifficultyAdjustmentInterval != 0 {
		// Not an adjustment block — keep current difficulty
		return tip.Header.Bits
	}

	// Fetch first block of the current retarget window
	windowStart := height - consensus.DifficultyAdjustmentInterval
	if windowStart < 0 {
		windowStart = 0
	}
	firstBlock, ok := c.byHeight[windowStart]
	if !ok {
		return tip.Header.Bits
	}

	return consensus.CalculateNextBits(
		tip.Header.Bits,
		firstBlock.Header.Timestamp,
		tip.Header.Timestamp,
	)
}

// Genesis returns the genesis block.
func (c *Chain) Genesis() *Block {
	return c.genesis
}

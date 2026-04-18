package blockchain

import (
	"errors"
	"fmt"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/consensus"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

// BlockValidator validates blocks against consensus rules.
type BlockValidator struct {
	chain *Chain
}

func NewBlockValidator(chain *Chain) *BlockValidator {
	return &BlockValidator{chain: chain}
}

// ValidateBlock performs full block validation.
func (v *BlockValidator) ValidateBlock(block *Block) error {
	if err := v.validateHeader(block); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if err := v.validateCoinbase(block); err != nil {
		return fmt.Errorf("coinbase: %w", err)
	}
	if err := v.validateTransactions(block); err != nil {
		return fmt.Errorf("transactions: %w", err)
	}
	if err := v.validateMerkleRoot(block); err != nil {
		return fmt.Errorf("merkle: %w", err)
	}
	if err := v.validateSize(block); err != nil {
		return fmt.Errorf("size: %w", err)
	}
	return nil
}

func (v *BlockValidator) validateHeader(block *Block) error {
	h := &block.Header

	// Version check
	if h.Version == 0 {
		return errors.New("version cannot be zero")
	}

	// Timestamp must be within 2 hours of now (clock skew tolerance)
	now := time.Now().Unix()
	if h.Timestamp > now+7200 {
		return fmt.Errorf("timestamp too far in the future: %d", h.Timestamp)
	}
	if h.Timestamp < now-86400*30 {
		return errors.New("timestamp too old")
	}

	// Validate PoW: block hash must satisfy difficulty
	blockHash := h.Hash()
	target, err := consensus.BitsToTarget(h.Bits)
	if err != nil {
		return fmt.Errorf("invalid bits: %w", err)
	}
	if !consensus.HashMeetsDifficulty(blockHash, target) {
		return errors.New("block hash does not meet difficulty target")
	}

	// Height must be prev+1
	if h.Height > 0 {
		prev, err := v.chain.GetBlockByHash(h.PrevHash)
		if err != nil {
			return fmt.Errorf("unknown previous block: %w", err)
		}
		if h.Height != prev.Header.Height+1 {
			return fmt.Errorf("invalid height: expected %d got %d",
				prev.Header.Height+1, h.Height)
		}
		if h.Timestamp <= prev.Header.Timestamp {
			return errors.New("timestamp must be after previous block")
		}
	}

	return nil
}

func (v *BlockValidator) validateCoinbase(block *Block) error {
	if block.Coinbase == nil {
		return errors.New("missing coinbase")
	}
	cb := block.Coinbase
	if cb.Height != block.Header.Height {
		return fmt.Errorf("coinbase height mismatch: %d vs %d", cb.Height, block.Header.Height)
	}

	expectedReward := config.HalvingSchedule(block.Header.Height)
	if cb.Reward > expectedReward {
		return fmt.Errorf("coinbase reward %d exceeds allowed %d", cb.Reward, expectedReward)
	}
	if cb.Reward < 0 {
		return errors.New("coinbase reward cannot be negative")
	}
	if err := wallet.ValidateAddress(cb.MinerAddress); err != nil {
		return fmt.Errorf("coinbase miner address: %w", err)
	}
	return nil
}

func (v *BlockValidator) validateTransactions(block *Block) error {
	if len(block.Transactions) > 100000 {
		return errors.New("too many transactions in block")
	}
	for i, tx := range block.Transactions {
		if err := tx.Validate(); err != nil {
			return fmt.Errorf("tx[%d]: %w", i, err)
		}
	}
	return nil
}

func (v *BlockValidator) validateMerkleRoot(block *Block) error {
	computed := block.ComputeMerkleRoot()
	if computed != block.Header.MerkleRoot {
		return fmt.Errorf("merkle root mismatch: got %s want %s",
			computed.String(), block.Header.MerkleRoot.String())
	}
	return nil
}

func (v *BlockValidator) validateSize(block *Block) error {
	size := block.Size()
	if size > config.MaxBlockSizeMax {
		return fmt.Errorf("block size %d exceeds maximum %d", size, config.MaxBlockSizeMax)
	}
	return nil
}

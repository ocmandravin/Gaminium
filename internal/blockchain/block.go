package blockchain

import (
	"encoding/binary"
	"time"

	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

// BlockHeader contains the consensus-critical metadata for a block.
type BlockHeader struct {
	Version        uint32
	Height         int64
	PrevHash       crypto.Hash
	MerkleRoot     crypto.Hash
	Timestamp      int64
	Bits           uint32 // compact difficulty target
	Nonce          uint64
	ExtraNonce     uint64
	OracleDataHash crypto.Hash // hash of oracle price data committed at this block
}

// Block is a complete GAMINIUM blockchain block.
type Block struct {
	Header       BlockHeader
	Transactions []*wallet.Transaction
	Coinbase     *CoinbaseTx
}

// CoinbaseTx is the miner reward transaction.
type CoinbaseTx struct {
	Height       int64
	MinerAddress string
	Reward       int64 // in Minium
	ExtraData    []byte
}

// Hash computes the block header hash using BLAKE3-512.
func (h *BlockHeader) Hash() crypto.Hash {
	return crypto.HashMany(h.Bytes())
}

// Bytes serialises the block header deterministically for hashing/PoW.
func (h *BlockHeader) Bytes() []byte {
	buf := make([]byte, 4+8+64+64+8+4+8+8+64)
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], h.Version)
	offset += 4
	binary.BigEndian.PutUint64(buf[offset:], uint64(h.Height))
	offset += 8
	copy(buf[offset:], h.PrevHash[:])
	offset += 64
	copy(buf[offset:], h.MerkleRoot[:])
	offset += 64
	binary.BigEndian.PutUint64(buf[offset:], uint64(h.Timestamp))
	offset += 8
	binary.BigEndian.PutUint32(buf[offset:], h.Bits)
	offset += 4
	binary.BigEndian.PutUint64(buf[offset:], h.Nonce)
	offset += 8
	binary.BigEndian.PutUint64(buf[offset:], h.ExtraNonce)
	offset += 8
	copy(buf[offset:], h.OracleDataHash[:])

	return buf
}

// MerkleRoot computes the merkle root of all transactions in the block.
func (b *Block) ComputeMerkleRoot() crypto.Hash {
	hashes := make([]crypto.Hash, 0, len(b.Transactions)+1)

	// Include coinbase as first element
	if b.Coinbase != nil {
		hashes = append(hashes, b.coinbaseHash())
	}

	for _, tx := range b.Transactions {
		hashes = append(hashes, tx.TxID())
	}

	return crypto.MerkleRoot(hashes)
}

func (b *Block) coinbaseHash() crypto.Hash {
	buf := make([]byte, 8+8)
	binary.BigEndian.PutUint64(buf[0:], uint64(b.Coinbase.Height))
	binary.BigEndian.PutUint64(buf[8:], uint64(b.Coinbase.Reward))
	return crypto.HashMany(
		[]byte("coinbase"),
		buf,
		[]byte(b.Coinbase.MinerAddress),
		b.Coinbase.ExtraData,
	)
}

// Size returns the approximate serialised block size in bytes.
func (b *Block) Size() int {
	size := 160 // header
	size += 100 // coinbase approx
	size += len(b.Transactions) * 800 // tx average
	return size
}

// NewBlock creates a new block with the given parameters.
func NewBlock(height int64, prevHash crypto.Hash, bits uint32) *Block {
	return &Block{
		Header: BlockHeader{
			Version:   1,
			Height:    height,
			PrevHash:  prevHash,
			Timestamp: time.Now().Unix(),
			Bits:      bits,
		},
	}
}

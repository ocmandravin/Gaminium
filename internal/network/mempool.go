package network

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

// MempoolEntry wraps a transaction with metadata for priority ordering.
type MempoolEntry struct {
	Tx        *wallet.Transaction
	TxID      crypto.Hash
	FeeRate   int64 // Minium per byte
	ReceivedAt time.Time
	Size      int
}

// Mempool manages the pending transaction pool.
// Max size: 300MB, expiry: 72 hours.
type Mempool struct {
	mu          sync.RWMutex
	entries     map[crypto.Hash]*MempoolEntry
	totalBytes  int
	maxBytes    int
	maxAge      time.Duration
}

// NewMempool creates a new mempool with default config.
func NewMempool() *Mempool {
	return &Mempool{
		entries:  make(map[crypto.Hash]*MempoolEntry),
		maxBytes: config.MempoolMaxSize,
		maxAge:   config.MempoolExpiry,
	}
}

// Add validates and inserts a transaction into the mempool.
func (m *Mempool) Add(tx *wallet.Transaction) error {
	if tx == nil {
		return errors.New("mempool: nil transaction")
	}
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("mempool: invalid tx: %w", err)
	}

	txID := tx.TxID()

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[txID]; exists {
		return errors.New("mempool: transaction already in pool")
	}

	// Estimate size (simplified)
	size := estimateTxSize(tx)

	// Check mempool capacity
	if m.totalBytes+size > m.maxBytes {
		m.evictLowestFee()
		if m.totalBytes+size > m.maxBytes {
			return errors.New("mempool: pool full — fee too low to replace existing transactions")
		}
	}

	entry := &MempoolEntry{
		Tx:         tx,
		TxID:       txID,
		FeeRate:    feeRate(tx, size),
		ReceivedAt: time.Now(),
		Size:       size,
	}

	m.entries[txID] = entry
	m.totalBytes += size
	return nil
}

// Remove removes a transaction from the pool (after inclusion in a block).
func (m *Mempool) Remove(txID crypto.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.entries[txID]; ok {
		m.totalBytes -= entry.Size
		delete(m.entries, txID)
	}
}

// Get retrieves a transaction by ID.
func (m *Mempool) Get(txID crypto.Hash) (*wallet.Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[txID]
	if !ok {
		return nil, false
	}
	return entry.Tx, true
}

// SelectForBlock returns the highest-fee transactions that fit in a block.
func (m *Mempool) SelectForBlock(maxBlockBytes int) []*wallet.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect and sort by fee rate (descending)
	entries := make([]*MempoolEntry, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}
	sortByFeeRate(entries)

	var selected []*wallet.Transaction
	usedBytes := 0
	for _, e := range entries {
		if usedBytes+e.Size > maxBlockBytes {
			continue
		}
		selected = append(selected, e.Tx)
		usedBytes += e.Size
	}
	return selected
}

// ExpireOld removes transactions that have been in the pool past expiry.
func (m *Mempool) ExpireOld() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.maxAge)
	removed := 0
	for id, entry := range m.entries {
		if entry.ReceivedAt.Before(cutoff) {
			m.totalBytes -= entry.Size
			delete(m.entries, id)
			removed++
		}
	}
	return removed
}

// Count returns the number of transactions in the pool.
func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// Size returns total bytes currently in the pool.
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalBytes
}

func (m *Mempool) evictLowestFee() {
	// Find and remove the lowest fee rate entry
	var lowest *MempoolEntry
	var lowestID crypto.Hash
	for id, e := range m.entries {
		if lowest == nil || e.FeeRate < lowest.FeeRate {
			lowest = e
			lowestID = id
		}
	}
	if lowest != nil {
		m.totalBytes -= lowest.Size
		delete(m.entries, lowestID)
	}
}

func estimateTxSize(tx *wallet.Transaction) int {
	// Rough estimate: 100 bytes overhead + inputs + outputs
	size := 100
	size += len(tx.Inputs) * 2500  // Dilithium sig ~2420 bytes + pubkey + metadata
	size += len(tx.Outputs) * 50
	return size
}

func feeRate(tx *wallet.Transaction, size int) int64 {
	if size == 0 {
		return 0
	}
	return tx.Fee / int64(size)
}

func sortByFeeRate(entries []*MempoolEntry) {
	// Insertion sort (good enough for typical mempool sizes during selection)
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].FeeRate > entries[j-1].FeeRate; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

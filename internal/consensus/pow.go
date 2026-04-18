package consensus

import (
	"context"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ocamndravin/gaminium/internal/crypto"
)

// MineResult holds the result of a successful PoW solution.
type MineResult struct {
	Nonce      uint64
	ExtraNonce uint64
	Hash       crypto.Hash
}

// RandomXMine simulates RandomX CPU mining using BLAKE3 as the PoW function
// during development. Production will integrate the RandomX library via cgo.
//
// Mining uses parallel goroutines across all CPU cores.
func RandomXMine(ctx context.Context, headerBytes []byte, bits uint32) (*MineResult, error) {
	target, err := BitsToTarget(bits)
	if err != nil {
		return nil, err
	}

	numWorkers := runtime.NumCPU()
	results := make(chan *MineResult, 1)
	var found atomic.Bool

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			mineWorker(ctx, headerBytes, target, uint64(workerID), uint64(numWorkers), &found, results)
		}(w)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	select {
	case result, ok := <-results:
		if ok && result != nil {
			return result, nil
		}
		return nil, context.Canceled
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func mineWorker(
	ctx context.Context,
	headerBytes []byte,
	target *big.Int,
	startNonce, step uint64,
	found *atomic.Bool,
	results chan<- *MineResult,
) {
	nonce := startNonce
	buf := make([]byte, len(headerBytes)+8)
	copy(buf, headerBytes)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if found.Load() {
			return
		}

		// Write nonce into last 8 bytes
		buf[len(headerBytes)] = byte(nonce >> 56)
		buf[len(headerBytes)+1] = byte(nonce >> 48)
		buf[len(headerBytes)+2] = byte(nonce >> 40)
		buf[len(headerBytes)+3] = byte(nonce >> 32)
		buf[len(headerBytes)+4] = byte(nonce >> 24)
		buf[len(headerBytes)+5] = byte(nonce >> 16)
		buf[len(headerBytes)+6] = byte(nonce >> 8)
		buf[len(headerBytes)+7] = byte(nonce)

		hash := crypto.HashBytes(buf)

		if HashMeetsDifficulty(hash, target) {
			if found.CompareAndSwap(false, true) {
				results <- &MineResult{
					Nonce: nonce,
					Hash:  hash,
				}
			}
			return
		}

		nonce += step
	}
}

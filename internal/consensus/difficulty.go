package consensus

import (
	"errors"
	"math/big"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

const (
	// DifficultyAdjustmentInterval is the number of blocks between adjustments.
	// GAMINIUM adjusts every 2016 blocks (~1 week at 5min blocks).
	DifficultyAdjustmentInterval = int64(2016)

	// MaxTargetBits is the easiest difficulty (genesis / min difficulty)
	MaxTargetBits = uint32(0x1e0fffff)
)

var (
	// maxTarget is the big.Int representation of the easiest target
	maxTarget = bitsToTargetUnchecked(MaxTargetBits)

	// Clamp: difficulty adjustment limited to 4x up or 1/4 down per period
	maxRetargetFactor = big.NewInt(4)
)

// BitsToTarget converts compact bits format to a *big.Int target.
func BitsToTarget(bits uint32) (*big.Int, error) {
	if bits == 0 {
		return nil, errors.New("bits cannot be zero")
	}
	target := bitsToTargetUnchecked(bits)
	if target.Sign() <= 0 {
		return nil, errors.New("invalid bits: non-positive target")
	}
	return target, nil
}

// TargetToBits converts a *big.Int target to compact bits format.
func TargetToBits(target *big.Int) uint32 {
	if target == nil || target.Sign() <= 0 {
		return MaxTargetBits
	}
	b := target.Bytes()
	if len(b) == 0 {
		return MaxTargetBits
	}

	// Compact format: 1 byte exponent + 3 bytes mantissa
	exponent := uint32(len(b))
	var mantissa uint32
	if len(b) >= 3 {
		mantissa = uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
	} else if len(b) == 2 {
		mantissa = uint32(b[0])<<8 | uint32(b[1])
	} else {
		mantissa = uint32(b[0])
	}

	// Normalise: leading byte must be < 0x80
	if mantissa&0x800000 != 0 {
		mantissa >>= 8
		exponent++
	}

	return (exponent << 24) | mantissa
}

// HashMeetsDifficulty returns true if the hash satisfies the target.
func HashMeetsDifficulty(hash crypto.Hash, target *big.Int) bool {
	hashInt := new(big.Int).SetBytes(hash[:])
	return hashInt.Cmp(target) <= 0
}

// CalculateNextBits computes the next difficulty target.
// Uses the previous DifficultyAdjustmentInterval blocks' timestamps.
func CalculateNextBits(lastBits uint32, firstTime, lastTime int64) uint32 {
	// Expected time for DifficultyAdjustmentInterval blocks
	expectedSeconds := int64(config.BlockTime.Seconds()) * DifficultyAdjustmentInterval

	actualSeconds := lastTime - firstTime
	if actualSeconds <= 0 {
		actualSeconds = 1
	}

	// Clamp: 4x max speedup, 4x max slowdown
	if actualSeconds < expectedSeconds/4 {
		actualSeconds = expectedSeconds / 4
	}
	if actualSeconds > expectedSeconds*4 {
		actualSeconds = expectedSeconds * 4
	}

	prevTarget, err := BitsToTarget(lastBits)
	if err != nil {
		return MaxTargetBits
	}

	// new_target = prev_target * actual / expected
	newTarget := new(big.Int).Mul(prevTarget, big.NewInt(actualSeconds))
	newTarget.Div(newTarget, big.NewInt(expectedSeconds))

	// Never easier than genesis difficulty
	if newTarget.Cmp(maxTarget) > 0 {
		newTarget = maxTarget
	}

	return TargetToBits(newTarget)
}

func bitsToTargetUnchecked(bits uint32) *big.Int {
	exponent := bits >> 24
	mantissa := bits & 0x007fffff
	target := new(big.Int).SetInt64(int64(mantissa))
	if exponent <= 3 {
		target.Rsh(target, (3-uint(exponent))*8)
	} else {
		target.Lsh(target, (uint(exponent)-3)*8)
	}
	return target
}


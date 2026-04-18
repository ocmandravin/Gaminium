package oracle

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// OracleConsensus reaches a consensus price from all oracle submissions.
// Uses the median of all valid submissions; nodes deviating >10% lose stake.

type ConsensusResult struct {
	BlockHeight  int64
	MedianFloor  float64
	Submissions  int
	Accepted     int
	Slashed      []crypto.Hash // node IDs to be slashed
}

// ReachConsensus computes the median price floor from oracle submissions.
// Submissions deviating >10% from the median are rejected and their nodes slashed.
func (n *OracleNetwork) ReachConsensus(blockHeight int64) (*ConsensusResult, error) {
	subs := n.GetSubmissions(blockHeight)

	if len(subs) == 0 {
		return nil, errors.New("oracle consensus: no submissions for this block")
	}
	if len(subs) < config.OracleMinNodes/2+1 {
		return nil, fmt.Errorf("oracle consensus: insufficient submissions (%d, need %d)",
			len(subs), config.OracleMinNodes/2+1)
	}

	// Extract floor values
	values := make([]float64, len(subs))
	for i, s := range subs {
		values[i] = s.FloorUSD
	}

	// Compute median
	medianFloor := floatMedian(values)
	if medianFloor <= 0 {
		return nil, errors.New("oracle consensus: invalid median floor (≤0)")
	}

	// Identify outliers: nodes deviating >10% from median
	result := &ConsensusResult{
		BlockHeight: blockHeight,
		MedianFloor: medianFloor,
		Submissions: len(subs),
	}

	for _, sub := range subs {
		deviation := math.Abs(sub.FloorUSD-medianFloor) / medianFloor
		if deviation > config.OracleDeviationLimit {
			result.Slashed = append(result.Slashed, sub.NodeID)
		} else {
			result.Accepted++
		}
	}

	return result, nil
}

// ApplySlashing slashes oracle nodes that submitted outlier prices.
// Each slashed node loses their entire stake (sent to treasury).
func (n *OracleNetwork) ApplySlashing(result *ConsensusResult) []int64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	var slashedAmounts []int64
	for _, nodeID := range result.Slashed {
		if node, ok := n.nodes[nodeID]; ok {
			slashedAmounts = append(slashedAmounts, node.StakeMinium)
			node.StakeMinium = 0
			node.Active = false
			// Remove from country count
			n.countryCount[node.Country]--
			delete(n.nodes, nodeID)
		}
	}
	return slashedAmounts
}

func floatMedian(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

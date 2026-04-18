package oracle

import (
	"errors"
	"fmt"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// StakeManager handles oracle node staking and unstaking logic.
type StakeManager struct {
	network  *OracleNetwork
	treasury *TreasuryAccount
}

// TreasuryAccount is a simplified treasury representation for stake slashing receipts.
type TreasuryAccount struct {
	BalanceMinium int64
}

func NewStakeManager(network *OracleNetwork) *StakeManager {
	return &StakeManager{
		network:  network,
		treasury: &TreasuryAccount{},
	}
}

// Stake registers an oracle node with a stake deposit.
// Verifies minimum stake requirement and country limits.
func (s *StakeManager) Stake(node *OracleNode) error {
	if node.StakeMinium < config.OracleStakeRequired {
		return fmt.Errorf("stake: minimum %d Minium required, got %d",
			config.OracleStakeRequired, node.StakeMinium)
	}
	return s.network.RegisterNode(node)
}

// Unstake allows a node to withdraw stake after a 30-day lockup.
// (lockup enforcement is handled at the blockchain level via time-lock)
func (s *StakeManager) Unstake(nodeID crypto.Hash) (int64, error) {
	s.network.mu.Lock()
	defer s.network.mu.Unlock()

	node, ok := s.network.nodes[nodeID]
	if !ok {
		return 0, errors.New("stake: node not found")
	}

	amount := node.StakeMinium
	s.network.countryCount[node.Country]--
	delete(s.network.nodes, nodeID)
	return amount, nil
}

// ProcessSlashing executes slashing for consensus violations.
// Slashed stake is sent to the treasury.
func (s *StakeManager) ProcessSlashing(result *ConsensusResult) int64 {
	slashedAmounts := s.network.ApplySlashing(result)
	total := int64(0)
	for _, amt := range slashedAmounts {
		total += amt
	}
	s.treasury.BalanceMinium += total
	return total
}

// ValidateStakeAmount checks if a stake amount meets requirements.
func ValidateStakeAmount(amount int64) error {
	if amount < config.OracleStakeRequired {
		return fmt.Errorf("minimum oracle stake is %d GMN (%d Minium)",
			config.OracleStakeRequired/config.MiniumPerGMN,
			config.OracleStakeRequired)
	}
	return nil
}

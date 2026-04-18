package governance

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// LockPeriod represents a vGMN time-lock duration.
type LockPeriod int

const (
	Lock3Months LockPeriod = iota // 1x weight
	Lock1Year                     // 4x weight
	Lock4Years                    // 8x weight
)

func (l LockPeriod) Weight() float64 {
	switch l {
	case Lock3Months:
		return float64(config.VoteWeightLock3M)
	case Lock1Year:
		return float64(config.VoteWeightLock1Y)
	case Lock4Years:
		return float64(config.VoteWeightLock4Y)
	}
	return 1.0
}

func (l LockPeriod) Duration() time.Duration {
	switch l {
	case Lock3Months:
		return 90 * 24 * time.Hour
	case Lock1Year:
		return 365 * 24 * time.Hour
	case Lock4Years:
		return 4 * 365 * 24 * time.Hour
	}
	return 90 * 24 * time.Hour
}

// VGMNLock represents a locked token position that grants voting power.
type VGMNLock struct {
	Owner      string // GMN1... address
	AmountMinium int64
	Period     LockPeriod
	LockedAt   time.Time
	UnlockAt   time.Time
}

// VotingPower returns weighted voting power for this lock.
func (v *VGMNLock) VotingPower() float64 {
	return float64(v.AmountMinium) * v.Period.Weight()
}

// Vote represents a single vote cast on a proposal.
type Vote struct {
	Voter     string // GMN1... address
	ProposalID crypto.Hash
	InFavor   bool
	Weight    float64
	Signature crypto.DilithiumSignature
	CastAt    time.Time
}

// VotingEngine manages the voting process for proposals.
type VotingEngine struct {
	mu       sync.RWMutex
	locks    map[string][]*VGMNLock        // address → locks
	votes    map[crypto.Hash][]*Vote        // proposalID → votes
	registry *ProposalRegistry
	// totalSupplyMinium used for % cap enforcement
	totalSupplyMinium int64
}

// NewVotingEngine creates a voting engine.
func NewVotingEngine(registry *ProposalRegistry, totalSupplyMinium int64) *VotingEngine {
	return &VotingEngine{
		locks:             make(map[string][]*VGMNLock),
		votes:             make(map[crypto.Hash][]*Vote),
		registry:          registry,
		totalSupplyMinium: totalSupplyMinium,
	}
}

// LockTokens creates a vGMN lock for an address.
func (e *VotingEngine) LockTokens(lock *VGMNLock) error {
	if lock == nil || lock.AmountMinium <= 0 {
		return errors.New("voting: invalid lock amount")
	}
	if lock.Owner == "" {
		return errors.New("voting: lock must have owner address")
	}
	lock.UnlockAt = lock.LockedAt.Add(lock.Period.Duration())

	e.mu.Lock()
	defer e.mu.Unlock()
	e.locks[lock.Owner] = append(e.locks[lock.Owner], lock)
	return nil
}

// CastVote casts a vote on a proposal.
func (e *VotingEngine) CastVote(vote *Vote) error {
	if vote == nil {
		return errors.New("voting: nil vote")
	}

	proposal, err := e.registry.Get(vote.ProposalID)
	if err != nil {
		return err
	}
	if proposal.Status != ProposalActive {
		return fmt.Errorf("voting: proposal is %s, not active", proposal.Status)
	}
	if time.Now().After(proposal.VotingEnds) {
		return errors.New("voting: voting period has ended")
	}

	// Compute voting weight for this address
	power := e.votingPower(vote.Voter)
	if power == 0 {
		return errors.New("voting: no locked tokens — lock vGMN to vote")
	}

	// Enforce max 5% per address per vote
	totalPower := e.totalVotingPower()
	if totalPower > 0 {
		fraction := power / totalPower
		if fraction > config.MaxVotePerAddress {
			// Cap at MaxVotePerAddress
			power = totalPower * config.MaxVotePerAddress
		}
	}

	vote.Weight = power
	vote.CastAt = time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.votes[vote.ProposalID] = append(e.votes[vote.ProposalID], vote)
	return nil
}

// Tally computes the vote result for a proposal.
func (e *VotingEngine) Tally(proposalID crypto.Hash) (forPct, againstPct float64, passed bool, err error) {
	proposal, err := e.registry.Get(proposalID)
	if err != nil {
		return 0, 0, false, err
	}

	e.mu.RLock()
	votes := e.votes[proposalID]
	e.mu.RUnlock()

	forWeight := 0.0
	totalWeight := 0.0
	for _, v := range votes {
		totalWeight += v.Weight
		if v.InFavor {
			forWeight += v.Weight
		}
	}

	if totalWeight == 0 {
		return 0, 0, false, nil
	}

	forPct = forWeight / totalWeight * 100
	againstPct = 100 - forPct

	threshold := proposalThreshold(proposal.Type)
	passed = forPct >= threshold
	return
}

// proposalThreshold returns the % required to pass by type.
func proposalThreshold(pt ProposalType) float64 {
	switch pt {
	case ProposalStandard:
		return 51.0
	case ProposalMajor:
		return 67.0
	case ProposalConstitutional:
		return 80.0
	case ProposalEmergency:
		return 75.0
	}
	return 67.0
}

func (e *VotingEngine) votingPower(address string) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	total := 0.0
	for _, lock := range e.locks[address] {
		if time.Now().Before(lock.UnlockAt) {
			total += lock.VotingPower()
		}
	}
	return total
}

func (e *VotingEngine) totalVotingPower() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	total := 0.0
	for _, locks := range e.locks {
		for _, lock := range locks {
			if time.Now().Before(lock.UnlockAt) {
				total += lock.VotingPower()
			}
		}
	}
	return total
}

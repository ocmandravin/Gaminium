package governance

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/internal/crypto"
)

// ProposalType defines the category of a governance proposal.
type ProposalType int

const (
	ProposalStandard      ProposalType = iota // 51%, 30-day timelock
	ProposalMajor                             // 67%, 60-day timelock
	ProposalConstitutional                    // 80%, 90-day timelock
	ProposalEmergency                         // 75%, 24-hour timelock
)

func (t ProposalType) String() string {
	switch t {
	case ProposalStandard:
		return "STANDARD"
	case ProposalMajor:
		return "MAJOR"
	case ProposalConstitutional:
		return "CONSTITUTIONAL"
	case ProposalEmergency:
		return "EMERGENCY"
	}
	return "UNKNOWN"
}

// ProposalStatus tracks the lifecycle of a proposal.
type ProposalStatus int

const (
	ProposalActive   ProposalStatus = iota
	ProposalPassed
	ProposalRejected
	ProposalExpired
	ProposalExecuted
)

func (s ProposalStatus) String() string {
	switch s {
	case ProposalActive:
		return "ACTIVE"
	case ProposalPassed:
		return "PASSED"
	case ProposalRejected:
		return "REJECTED"
	case ProposalExpired:
		return "EXPIRED"
	case ProposalExecuted:
		return "EXECUTED"
	}
	return "UNKNOWN"
}

// ImmutableCore lists items that can never be changed by governance.
var ImmutableCore = []string{
	"total_supply",
	"halving_schedule",
	"price_floor_formula",
	"cryptographic_scheme",
}

// Proposal is a governance proposal for protocol changes.
type Proposal struct {
	ID          crypto.Hash
	Type        ProposalType
	Title       string
	Description string
	Proposer    string // GMN1... address
	Target      string // what is being changed
	CreatedAt   time.Time
	VotingEnds  time.Time
	TimelockEnd time.Time
	Status      ProposalStatus
}

// ProposalRegistry tracks all proposals.
type ProposalRegistry struct {
	mu        sync.RWMutex
	proposals map[crypto.Hash]*Proposal
}

// NewProposalRegistry creates an empty proposal registry.
func NewProposalRegistry() *ProposalRegistry {
	return &ProposalRegistry{
		proposals: make(map[crypto.Hash]*Proposal),
	}
}

// Submit adds a new proposal.
func (r *ProposalRegistry) Submit(p *Proposal) error {
	if p == nil {
		return errors.New("governance: nil proposal")
	}
	if err := validateProposal(p); err != nil {
		return err
	}

	// Reject changes to immutable core
	for _, item := range ImmutableCore {
		if p.Target == item {
			return fmt.Errorf("governance: %q is immutable and cannot be changed by proposal", item)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	p.ID = crypto.HashMany(
		[]byte("proposal"),
		[]byte(p.Title),
		[]byte(p.Proposer),
		[]byte(p.CreatedAt.String()),
	)
	r.proposals[p.ID] = p
	return nil
}

// Get returns a proposal by ID.
func (r *ProposalRegistry) Get(id crypto.Hash) (*Proposal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.proposals[id]
	if !ok {
		return nil, fmt.Errorf("governance: proposal %s not found", id.String()[:16])
	}
	return p, nil
}

// UpdateStatus sets the proposal status.
func (r *ProposalRegistry) UpdateStatus(id crypto.Hash, status ProposalStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.proposals[id]
	if !ok {
		return fmt.Errorf("governance: proposal %s not found", id.String()[:16])
	}
	p.Status = status
	return nil
}

func validateProposal(p *Proposal) error {
	if p.Title == "" {
		return errors.New("governance: proposal must have a title")
	}
	if p.Proposer == "" {
		return errors.New("governance: proposal must have a proposer address")
	}
	if p.Target == "" {
		return errors.New("governance: proposal must specify what it changes")
	}
	return nil
}

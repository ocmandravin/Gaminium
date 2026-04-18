package governance

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/internal/crypto"
)

// TimelockDurations maps proposal type to execution delay after passing.
var TimelockDurations = map[ProposalType]time.Duration{
	ProposalStandard:      30 * 24 * time.Hour,
	ProposalMajor:         60 * 24 * time.Hour,
	ProposalConstitutional: 90 * 24 * time.Hour,
	ProposalEmergency:     24 * time.Hour,
}

// TimelockQueue holds proposals that have passed voting and are awaiting execution.
type TimelockQueue struct {
	mu      sync.RWMutex
	pending map[crypto.Hash]*TimelockEntry
}

// TimelockEntry is a proposal waiting to execute after its timelock expires.
type TimelockEntry struct {
	ProposalID  crypto.Hash
	ExecuteAt   time.Time
	Type        ProposalType
	ExecutionFn func() error // the actual protocol change function
}

// NewTimelockQueue creates an empty timelock queue.
func NewTimelockQueue() *TimelockQueue {
	return &TimelockQueue{
		pending: make(map[crypto.Hash]*TimelockEntry),
	}
}

// Enqueue adds a passed proposal to the timelock queue.
func (q *TimelockQueue) Enqueue(proposal *Proposal, executionFn func() error) error {
	if proposal == nil {
		return errors.New("timelock: nil proposal")
	}
	if proposal.Status != ProposalPassed {
		return fmt.Errorf("timelock: only passed proposals can be enqueued, got %s", proposal.Status)
	}

	delay, ok := TimelockDurations[proposal.Type]
	if !ok {
		delay = 30 * 24 * time.Hour
	}

	entry := &TimelockEntry{
		ProposalID:  proposal.ID,
		ExecuteAt:   time.Now().Add(delay),
		Type:        proposal.Type,
		ExecutionFn: executionFn,
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending[proposal.ID] = entry
	return nil
}

// ExecuteReady runs all proposals whose timelock has expired.
func (q *TimelockQueue) ExecuteReady(registry *ProposalRegistry) []error {
	q.mu.Lock()
	defer q.mu.Unlock()

	var errs []error
	for id, entry := range q.pending {
		if time.Now().Before(entry.ExecuteAt) {
			continue
		}
		if entry.ExecutionFn != nil {
			if err := entry.ExecutionFn(); err != nil {
				errs = append(errs, fmt.Errorf("timelock execute proposal %s: %w", id.String()[:16], err))
				continue
			}
		}
		delete(q.pending, id)
		registry.UpdateStatus(id, ProposalExecuted) //nolint: errcheck
	}
	return errs
}

// Cancel removes a proposal from the queue (emergency cancellation).
func (q *TimelockQueue) Cancel(proposalID crypto.Hash) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.pending[proposalID]; !ok {
		return fmt.Errorf("timelock: proposal %s not in queue", proposalID.String()[:16])
	}
	delete(q.pending, proposalID)
	return nil
}

// TimeRemaining returns how long until a queued proposal can execute.
func (q *TimelockQueue) TimeRemaining(proposalID crypto.Hash) (time.Duration, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	entry, ok := q.pending[proposalID]
	if !ok {
		return 0, fmt.Errorf("timelock: proposal %s not in queue", proposalID.String()[:16])
	}
	remaining := time.Until(entry.ExecuteAt)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

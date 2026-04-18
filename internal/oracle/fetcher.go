package oracle

import (
	"fmt"
	"time"

	"github.com/ocamndravin/gaminium/internal/ai"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/pricefloor"
	"github.com/ocamndravin/gaminium/internal/wallet"
)

// LocalFetcher is run by each oracle node locally to compute a price submission.
// Every node independently fetches all data and runs AI validation.

type LocalFetcher struct {
	nodeID      crypto.Hash
	signingKey  *wallet.DerivedKey
	calculator  *pricefloor.Calculator
	carbonZone  string // this node's country/grid zone
}

// NewLocalFetcher creates a fetcher for a specific oracle node.
func NewLocalFetcher(
	nodeID crypto.Hash,
	signingKey *wallet.DerivedKey,
	carbonZone string,
	aiValidator *ai.Validator,
) *LocalFetcher {
	calc := pricefloor.NewCalculator(aiValidator)
	return &LocalFetcher{
		nodeID:     nodeID,
		signingKey: signingKey,
		calculator: calc,
		carbonZone: carbonZone,
	}
}

// Fetch computes the local price floor estimate and returns a signed submission.
func (f *LocalFetcher) Fetch(blockHeight int64) (*PriceSubmission, error) {
	result, err := f.calculator.Update(blockHeight, f.carbonZone)
	if err != nil {
		return nil, fmt.Errorf("oracle fetcher: floor calculation failed: %w", err)
	}

	sub := &PriceSubmission{
		NodeID:      f.nodeID,
		BlockHeight: blockHeight,
		FloorUSD:    result.Floor,
		Inputs:      pricefloor.FloorInputs{}, // logged locally; not transmitted for privacy
		Timestamp:   time.Now(),
		PublicKey:   f.signingKey.DilithiumKey.Public,
	}

	// Sign the submission
	msg := submissionSigningBytes(sub)
	sig, err := crypto.DilithiumSign(f.signingKey.DilithiumKey.Private, msg)
	if err != nil {
		return nil, fmt.Errorf("oracle fetcher: signing failed: %w", err)
	}
	sub.Signature = sig

	return sub, nil
}

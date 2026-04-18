package oracle

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
	"github.com/ocamndravin/gaminium/internal/pricefloor"
)

// OracleNode represents a single oracle network participant.
type OracleNode struct {
	ID          crypto.Hash
	Address     string // network address
	Country     string // ISO country code
	StakeMinium int64  // staked GMN in Minium
	PublicKey   crypto.DilithiumPublicKey
	JoinedAt    time.Time
	Active      bool
}

// PriceSubmission is a signed price floor submission from an oracle node.
type PriceSubmission struct {
	NodeID      crypto.Hash
	BlockHeight int64
	FloorUSD    float64
	Inputs      pricefloor.FloorInputs
	Timestamp   time.Time
	Signature   crypto.DilithiumSignature
	PublicKey   crypto.DilithiumPublicKey
}

// OracleNetwork manages the full set of oracle nodes and reaches consensus on price floor.
type OracleNetwork struct {
	mu          sync.RWMutex
	nodes       map[crypto.Hash]*OracleNode
	submissions map[int64][]*PriceSubmission // blockHeight → submissions
	countryCount map[string]int
}

// NewOracleNetwork creates an empty oracle network.
func NewOracleNetwork() *OracleNetwork {
	return &OracleNetwork{
		nodes:        make(map[crypto.Hash]*OracleNode),
		submissions:  make(map[int64][]*PriceSubmission),
		countryCount: make(map[string]int),
	}
}

// RegisterNode registers a new oracle node after verifying stake and limits.
func (n *OracleNetwork) RegisterNode(node *OracleNode) error {
	if node == nil {
		return errors.New("oracle: nil node")
	}
	if node.StakeMinium < config.OracleStakeRequired {
		return fmt.Errorf("oracle: stake %d below minimum %d Minium",
			node.StakeMinium, config.OracleStakeRequired)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Enforce max 3 nodes per country
	if n.countryCount[node.Country] >= config.OracleMaxPerCountry {
		return fmt.Errorf("oracle: country %s already has %d/%d nodes",
			node.Country, n.countryCount[node.Country], config.OracleMaxPerCountry)
	}

	// Validate IP is real (basic sanity check)
	host, _, err := net.SplitHostPort(node.Address)
	if err != nil {
		host = node.Address
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("oracle: invalid node address: %s", node.Address)
	}

	n.nodes[node.ID] = node
	n.countryCount[node.Country]++
	return nil
}

// RemoveNode removes a node (e.g., after stake slashing).
func (n *OracleNetwork) RemoveNode(id crypto.Hash) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if node, ok := n.nodes[id]; ok {
		n.countryCount[node.Country]--
		delete(n.nodes, id)
	}
}

// NodeCount returns the number of active oracle nodes.
func (n *OracleNetwork) NodeCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.nodes)
}

// IsOperational returns true if the network has enough nodes to operate.
func (n *OracleNetwork) IsOperational() bool {
	return n.NodeCount() >= config.OracleMinNodes
}

// SubmitPrice accepts a price submission from an oracle node.
func (n *OracleNetwork) SubmitPrice(sub *PriceSubmission) error {
	if sub == nil {
		return errors.New("oracle: nil submission")
	}

	// Verify node is registered
	n.mu.RLock()
	_, exists := n.nodes[sub.NodeID]
	n.mu.RUnlock()
	if !exists {
		return fmt.Errorf("oracle: unknown node %s", sub.NodeID.String()[:16])
	}

	// Verify signature
	msg := submissionSigningBytes(sub)
	if err := crypto.DilithiumVerify(sub.PublicKey, msg, sub.Signature); err != nil {
		return fmt.Errorf("oracle: invalid submission signature: %w", err)
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.submissions[sub.BlockHeight] = append(n.submissions[sub.BlockHeight], sub)
	return nil
}

// GetSubmissions returns all price submissions for a block height.
func (n *OracleNetwork) GetSubmissions(blockHeight int64) []*PriceSubmission {
	n.mu.RLock()
	defer n.mu.RUnlock()
	subs := n.submissions[blockHeight]
	result := make([]*PriceSubmission, len(subs))
	copy(result, subs)
	return result
}

func submissionSigningBytes(sub *PriceSubmission) []byte {
	// Deterministic serialisation for signature verification
	buf := make([]byte, 8)
	_ = sub.BlockHeight
	buf[0] = byte(sub.BlockHeight >> 56)
	buf[1] = byte(sub.BlockHeight >> 48)
	buf[2] = byte(sub.BlockHeight >> 40)
	buf[3] = byte(sub.BlockHeight >> 32)
	buf[4] = byte(sub.BlockHeight >> 24)
	buf[5] = byte(sub.BlockHeight >> 16)
	buf[6] = byte(sub.BlockHeight >> 8)
	buf[7] = byte(sub.BlockHeight)

	floatBits := crypto.HashMany(
		[]byte("oracle-submission"),
		buf,
		sub.NodeID[:],
		[]byte(fmt.Sprintf("%.8f", sub.FloorUSD)),
	)
	return floatBits[:]
}

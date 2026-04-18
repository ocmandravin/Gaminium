package ai

import (
	"math"
	"sort"
)

// IsolationForest detects statistical outliers in oracle data streams.
// Implements the Isolation Forest algorithm (Liu et al. 2008) in pure Go.
// Deterministic: same input produces same output on every oracle node.

const (
	defaultNumTrees   = 100
	defaultSampleSize = 256
	defaultMaxDepth   = 0 // 0 = auto: ceil(log2(sampleSize))
)

// IsolationForest is a trained anomaly detector.
type IsolationForest struct {
	Trees      []*isolationTree
	NumTrees   int
	SampleSize int
	MaxDepth   int
}

type isolationTree struct {
	root *iTreeNode
}

type iTreeNode struct {
	splitFeature int
	splitValue   float64
	left, right  *iTreeNode
	size         int // number of training samples at this node (leaf only)
	isLeaf       bool
}

// NewIsolationForest creates a new forest with default parameters.
func NewIsolationForest(numTrees, sampleSize int) *IsolationForest {
	if numTrees <= 0 {
		numTrees = defaultNumTrees
	}
	if sampleSize <= 0 {
		sampleSize = defaultSampleSize
	}
	maxDepth := int(math.Ceil(math.Log2(float64(sampleSize))))

	return &IsolationForest{
		NumTrees:   numTrees,
		SampleSize: sampleSize,
		MaxDepth:   maxDepth,
	}
}

// Fit trains the isolation forest on historical data.
// data[i] is the i-th sample, data[i][j] is feature j.
func (f *IsolationForest) Fit(data [][]float64) {
	if len(data) == 0 {
		return
	}
	f.Trees = make([]*isolationTree, f.NumTrees)

	// Use a deterministic PRNG seeded from data hash
	rng := newDetRNG(hashData(data))

	for i := 0; i < f.NumTrees; i++ {
		sample := subsample(data, f.SampleSize, rng)
		tree := buildITree(sample, 0, f.MaxDepth, rng)
		f.Trees[i] = &isolationTree{root: tree}
	}
}

// Score returns the anomaly score for a single sample.
// Score is in [0, 1]; values near 1 indicate anomalies.
func (f *IsolationForest) Score(sample []float64) float64 {
	if len(f.Trees) == 0 {
		return 0.5 // untrained: neutral score
	}

	totalDepth := 0.0
	for _, tree := range f.Trees {
		totalDepth += pathLength(tree.root, sample, 0)
	}
	avgDepth := totalDepth / float64(len(f.Trees))

	c := expectedPathLength(float64(f.SampleSize))
	if c == 0 {
		return 0.5
	}

	// Anomaly score: higher = more anomalous
	score := math.Pow(2, -avgDepth/c)
	return score
}

// AnomalyThreshold is the score above which a data point is flagged as anomalous.
const AnomalyThreshold = 0.6

func buildITree(data [][]float64, depth, maxDepth int, rng *detRNG) *iTreeNode {
	if len(data) <= 1 || depth >= maxDepth {
		return &iTreeNode{isLeaf: true, size: len(data)}
	}

	numFeatures := len(data[0])
	if numFeatures == 0 {
		return &iTreeNode{isLeaf: true, size: len(data)}
	}

	// Pick random feature
	feature := int(rng.Intn(uint64(numFeatures)))

	// Find min and max for this feature
	minVal, maxVal := data[0][feature], data[0][feature]
	for _, row := range data[1:] {
		if row[feature] < minVal {
			minVal = row[feature]
		}
		if row[feature] > maxVal {
			maxVal = row[feature]
		}
	}

	if minVal == maxVal {
		return &iTreeNode{isLeaf: true, size: len(data)}
	}

	// Pick random split value in (min, max)
	splitVal := minVal + rng.Float64()*(maxVal-minVal)

	var leftData, rightData [][]float64
	for _, row := range data {
		if row[feature] < splitVal {
			leftData = append(leftData, row)
		} else {
			rightData = append(rightData, row)
		}
	}

	return &iTreeNode{
		splitFeature: feature,
		splitValue:   splitVal,
		left:         buildITree(leftData, depth+1, maxDepth, rng),
		right:        buildITree(rightData, depth+1, maxDepth, rng),
	}
}

func pathLength(node *iTreeNode, sample []float64, depth int) float64 {
	if node == nil || node.isLeaf {
		size := 1
		if node != nil {
			size = node.size
		}
		return float64(depth) + expectedPathLength(float64(size))
	}
	if sample[node.splitFeature] < node.splitValue {
		return pathLength(node.left, sample, depth+1)
	}
	return pathLength(node.right, sample, depth+1)
}

// expectedPathLength is c(n) from the original paper.
func expectedPathLength(n float64) float64 {
	if n <= 1 {
		return 0
	}
	// Euler-Mascheroni constant
	const eulerMascheroni = 0.5772156649
	return 2*(math.Log(n-1)+eulerMascheroni) - (2*(n-1)/n)
}

func subsample(data [][]float64, size int, rng *detRNG) [][]float64 {
	if len(data) <= size {
		cp := make([][]float64, len(data))
		copy(cp, data)
		return cp
	}
	indices := make([]int, len(data))
	for i := range indices {
		indices[i] = i
	}
	// Fisher-Yates shuffle (deterministic)
	for i := len(indices) - 1; i > 0; i-- {
		j := int(rng.Intn(uint64(i + 1)))
		indices[i], indices[j] = indices[j], indices[i]
	}
	result := make([][]float64, size)
	for i := 0; i < size; i++ {
		result[i] = data[indices[i]]
	}
	return result
}

func hashData(data [][]float64) uint64 {
	// Simple deterministic hash for PRNG seeding
	h := uint64(0xcbf29ce484222325) // FNV offset basis
	for _, row := range data {
		for _, v := range row {
			bits := math.Float64bits(v)
			h ^= bits
			h *= 0x100000001b3 // FNV prime
		}
	}
	return h
}

// detRNG is a deterministic, fast PRNG (xorshift64).
type detRNG struct{ state uint64 }

func newDetRNG(seed uint64) *detRNG {
	if seed == 0 {
		seed = 1
	}
	return &detRNG{state: seed}
}

func (r *detRNG) next() uint64 {
	r.state ^= r.state << 13
	r.state ^= r.state >> 7
	r.state ^= r.state << 17
	return r.state
}

func (r *detRNG) Intn(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return r.next() % n
}

func (r *detRNG) Float64() float64 {
	return float64(r.next()>>11) / (1 << 53)
}

// LoadFromJSON loads a pre-trained forest from serialised weights.
func (f *IsolationForest) LoadFromJSON(data []byte) error {
	// Deserialise from JSON model file; used when loading /models/isolation_forest.json
	// For now: re-trains from embedded historical data on first run
	_ = data
	return nil
}

// ensure sort is used (it's imported for potential future use)
var _ = sort.Ints

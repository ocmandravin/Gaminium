package ai

import "math"

// ConfidenceScore classifies an anomaly score into a confidence band.
type ConfidenceScore struct {
	Value    float64 // 0-100
	Band     ConfidenceBand
	Weight   float64 // weight to apply in price formula (0-1)
	Rejected bool
}

// ConfidenceBand represents the quality tier of a data point.
type ConfidenceBand int

const (
	BandHigh    ConfidenceBand = iota // 95-100: passes immediately
	BandMedium                        // 80-94: minor anomaly noted
	BandLow                           // 60-79: reduced weight
	BandRejected                      // <60: rejected
)

func (b ConfidenceBand) String() string {
	switch b {
	case BandHigh:
		return "HIGH"
	case BandMedium:
		return "MEDIUM"
	case BandLow:
		return "LOW"
	default:
		return "REJECTED"
	}
}

// Scorer combines Isolation Forest and LSTM scores into a final confidence score.
type Scorer struct {
	forest *IsolationForest
	lstm   *LSTMModel
}

// NewScorer creates a scorer from pre-trained models.
func NewScorer(forest *IsolationForest, lstm *LSTMModel) *Scorer {
	return &Scorer{forest: forest, lstm: lstm}
}

// ScoreDataPoint evaluates a new data point against historical context.
// historicalSeries: last 90 days of data for LSTM context.
// features: multi-dimensional feature vector for Isolation Forest.
func (s *Scorer) ScoreDataPoint(
	value float64,
	historicalSeries []float64,
	features []float64,
) ConfidenceScore {
	ifScore := 0.5
	if s.forest != nil && len(s.forest.Trees) > 0 {
		ifScore = s.forest.Score(features)
	}

	lstmScore := 0.5
	if s.lstm != nil && s.lstm.Trained && len(historicalSeries) >= 2 {
		score, err := s.lstm.AnomalyScore(historicalSeries, value)
		if err == nil {
			lstmScore = score
		}
	}

	// Combined anomaly score: weighted average (IF: 50%, LSTM: 50%)
	combinedAnomaly := 0.5*ifScore + 0.5*lstmScore

	// Confidence is inverse of anomaly
	confidence := (1.0 - combinedAnomaly) * 100.0
	confidence = math.Max(0, math.Min(100, confidence))

	return classifyConfidence(confidence)
}

func classifyConfidence(score float64) ConfidenceScore {
	cs := ConfidenceScore{Value: score}
	switch {
	case score >= 95:
		cs.Band = BandHigh
		cs.Weight = 1.0
	case score >= 80:
		cs.Band = BandMedium
		cs.Weight = 0.85
	case score >= 60:
		cs.Band = BandLow
		cs.Weight = 0.5
	default:
		cs.Band = BandRejected
		cs.Weight = 0
		cs.Rejected = true
	}
	return cs
}

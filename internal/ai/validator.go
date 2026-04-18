package ai

import (
	"fmt"
	"time"
)

// DataPoint represents a single oracle data submission.
type DataPoint struct {
	Source    string
	Value     float64
	Timestamp time.Time
	Features  []float64
}

// ValidationResult is the output of AI validation.
type ValidationResult struct {
	DataPoint  DataPoint
	Score      ConfidenceScore
	Accepted   bool
	Reason     string
}

// Validator applies AI models to oracle data before it enters the price formula.
type Validator struct {
	scorer  *Scorer
	history map[string][]float64 // source → rolling 90-day window
}

// NewValidator creates a validator backed by trained AI models.
func NewValidator(scorer *Scorer) *Validator {
	return &Validator{
		scorer:  scorer,
		history: make(map[string][]float64),
	}
}

// Validate scores a data point and decides accept/reject.
func (v *Validator) Validate(dp DataPoint) ValidationResult {
	history := v.history[dp.Source]

	score := v.scorer.ScoreDataPoint(dp.Value, history, dp.Features)

	result := ValidationResult{
		DataPoint: dp,
		Score:     score,
		Accepted:  !score.Rejected,
	}

	switch score.Band {
	case BandHigh:
		result.Reason = "passes: high confidence"
	case BandMedium:
		result.Reason = fmt.Sprintf("passes with note: medium confidence (%.1f%%)", score.Value)
	case BandLow:
		result.Reason = fmt.Sprintf("reduced weight: low confidence (%.1f%%)", score.Value)
	default:
		result.Reason = fmt.Sprintf("rejected: confidence %.1f%% below threshold", score.Value)
	}

	// Update rolling history (keep last 90 days = 90 samples at daily granularity)
	if result.Accepted {
		history = append(history, dp.Value)
		if len(history) > 90 {
			history = history[len(history)-90:]
		}
		v.history[dp.Source] = history
	}

	return result
}

// ValidateMultiple validates a batch of data points and returns accepted ones with weights.
func (v *Validator) ValidateMultiple(points []DataPoint) []ValidationResult {
	results := make([]ValidationResult, 0, len(points))
	for _, dp := range points {
		results = append(results, v.Validate(dp))
	}
	return results
}

// WeightedMedian computes the weighted median of accepted data points.
func WeightedMedian(results []ValidationResult) (float64, error) {
	type weightedVal struct {
		value  float64
		weight float64
	}

	var vals []weightedVal
	totalWeight := 0.0

	for _, r := range results {
		if !r.Accepted {
			continue
		}
		vals = append(vals, weightedVal{r.DataPoint.Value, r.Score.Weight})
		totalWeight += r.Score.Weight
	}

	if len(vals) == 0 || totalWeight == 0 {
		return 0, fmt.Errorf("ai: no accepted data points for median computation")
	}

	// Sort by value
	for i := 1; i < len(vals); i++ {
		for j := i; j > 0 && vals[j].value < vals[j-1].value; j-- {
			vals[j], vals[j-1] = vals[j-1], vals[j]
		}
	}

	// Find weighted median
	cumWeight := 0.0
	half := totalWeight / 2.0
	for _, v := range vals {
		cumWeight += v.weight
		if cumWeight >= half {
			return v.value, nil
		}
	}

	return vals[len(vals)-1].value, nil
}

package pricefloor

import "math"

// FloorFormula implements the GAMINIUM price floor calculation:
//
//   FLOOR = BaseUnitCost × EnergyIndex × CleanMultiplier × ScarcityFactor × VolumeFactor
//
// Pure energy basis — no commodities, no gold, no oil.
// All factors are dimensionless multipliers; FLOOR is expressed in USD per GMN.

const (
	// BaseUnitCost: minimum cost to produce 1 GMN in USD.
	// Represents ~30 minutes of average global energy cost at 100W mining rig.
	BaseUnitCost = 0.10 // USD

	// VolumeFactorCap caps the volume boost at 3x.
	VolumeFactorCap = 3.0

	// ScarcityFactorMin: when circulating = total supply, scarcity = 1.0.
	ScarcityFactorMin = 1.0

	// ScarcityFactorMax: at genesis (near zero circulation), scarcity = 2.0.
	ScarcityFactorMax = 2.0
)

// FloorInputs holds all the inputs needed to compute the price floor.
type FloorInputs struct {
	// Energy
	EnergyUSDkWh float64 // USD/kWh (72h rolling average of government data)

	// Clean energy multiplier (1.0–1.5)
	CleanEnergyMultiplier float64

	// Supply
	TotalSupplyMinium      int64
	CirculatingMinium      int64

	// Volume
	DailyTxVolumeMinium    int64 // total GMN transacted in last 24h (in Minium)
	BaselineTxVolumeMinium int64 // 30-day average daily volume (in Minium)
}

// FloorResult holds the computed price floor and intermediate factors.
type FloorResult struct {
	Floor                 float64 // USD per GMN
	BaseUnitCost          float64
	EnergyFactor          float64
	CleanEnergyMultiplier float64
	ScarcityFactor        float64
	VolumeFactor          float64
}

// Calculate computes the GAMINIUM price floor given all inputs.
func Calculate(inputs FloorInputs) FloorResult {
	// Energy factor: normalise USD/kWh against base energy cost (0.10 USD/kWh global avg 2024)
	const baseEnergyUSDkWh = 0.10
	energyFactor := inputs.EnergyUSDkWh / baseEnergyUSDkWh
	energyFactor = math.Max(0.5, math.Min(5.0, energyFactor)) // clamp to [0.5, 5.0]

	// Clean energy multiplier: already in [1.0, 1.5]
	cleanMult := inputs.CleanEnergyMultiplier
	cleanMult = math.Max(CleanMultiplierMin, math.Min(CleanMultiplierMax, cleanMult))

	// Scarcity factor: total / circulating, clamped to [1.0, 2.0]
	scarcityFactor := ScarcityFactorMin
	if inputs.CirculatingMinium > 0 && inputs.TotalSupplyMinium > 0 {
		ratio := float64(inputs.TotalSupplyMinium) / float64(inputs.CirculatingMinium)
		scarcityFactor = math.Max(ScarcityFactorMin, math.Min(ScarcityFactorMax, ratio))
	}

	// Volume factor: daily volume / baseline, capped at 3x
	volumeFactor := 1.0
	if inputs.BaselineTxVolumeMinium > 0 {
		rawFactor := float64(inputs.DailyTxVolumeMinium) / float64(inputs.BaselineTxVolumeMinium)
		volumeFactor = math.Max(0.5, math.Min(VolumeFactorCap, rawFactor))
	}

	floor := BaseUnitCost * energyFactor * cleanMult * scarcityFactor * volumeFactor

	return FloorResult{
		Floor:                 floor,
		BaseUnitCost:          BaseUnitCost,
		EnergyFactor:          energyFactor,
		CleanEnergyMultiplier: cleanMult,
		ScarcityFactor:        scarcityFactor,
		VolumeFactor:          volumeFactor,
	}
}

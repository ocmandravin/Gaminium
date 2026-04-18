package tests

import (
	"testing"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/pricefloor"
)

func TestFloorFormulaBaseline(t *testing.T) {
	// All factors at 1.0 (neutral) — floor should equal BaseUnitCost
	inputs := pricefloor.FloorInputs{
		EnergyUSDkWh:           0.10, // exactly base energy
		CleanEnergyMultiplier:  1.0,
		TotalSupplyMinium:      config.TotalSupplyMinum,
		CirculatingMinium:      config.TotalSupplyMinum, // fully circulating → scarcity = 1.0
		DailyTxVolumeMinium:    1000 * config.MiniumPerGMN,
		BaselineTxVolumeMinium: 1000 * config.MiniumPerGMN, // ratio = 1.0
	}

	result := pricefloor.Calculate(inputs)

	if result.Floor <= 0 {
		t.Error("floor should be positive")
	}
	// With all factors ~1.0, floor ≈ BaseUnitCost
	if result.Floor < 0.05 || result.Floor > 1.0 {
		t.Errorf("baseline floor out of expected range: $%.4f", result.Floor)
	}
}

func TestFloorScarcityFactor(t *testing.T) {
	// Low circulation → high scarcity
	inputs := pricefloor.FloorInputs{
		EnergyUSDkWh:           0.10,
		CleanEnergyMultiplier:  1.0,
		TotalSupplyMinium:      config.TotalSupplyMinum,
		CirculatingMinium:      config.TotalSupplyMinum / 10, // only 10% circulating
		DailyTxVolumeMinium:    1000 * config.MiniumPerGMN,
		BaselineTxVolumeMinium: 1000 * config.MiniumPerGMN,
	}

	high := pricefloor.Calculate(inputs)

	// Full circulation
	inputs.CirculatingMinium = config.TotalSupplyMinum
	low := pricefloor.Calculate(inputs)

	if high.Floor <= low.Floor {
		t.Errorf("low circulation should produce higher floor: %.4f vs %.4f", high.Floor, low.Floor)
	}
}

func TestFloorCleanEnergyMultiplier(t *testing.T) {
	base := pricefloor.FloorInputs{
		EnergyUSDkWh:           0.10,
		CirculatingMinium:      config.TotalSupplyMinum,
		TotalSupplyMinium:      config.TotalSupplyMinum,
		DailyTxVolumeMinium:    1000 * config.MiniumPerGMN,
		BaselineTxVolumeMinium: 1000 * config.MiniumPerGMN,
	}

	base.CleanEnergyMultiplier = 1.0
	dirty := pricefloor.Calculate(base)

	base.CleanEnergyMultiplier = 1.5
	clean := pricefloor.Calculate(base)

	if clean.Floor <= dirty.Floor {
		t.Errorf("clean energy should produce higher floor: clean=%.4f dirty=%.4f",
			clean.Floor, dirty.Floor)
	}
}

func TestFloorVolumeCapAt3x(t *testing.T) {
	inputs := pricefloor.FloorInputs{
		EnergyUSDkWh:           0.10,
		CleanEnergyMultiplier:  1.0,
		CirculatingMinium:      config.TotalSupplyMinum,
		TotalSupplyMinium:      config.TotalSupplyMinum,
		BaselineTxVolumeMinium: 1000 * config.MiniumPerGMN,
		DailyTxVolumeMinium:    1000000 * config.MiniumPerGMN, // 1000x baseline
	}

	result := pricefloor.Calculate(inputs)
	if result.VolumeFactor > pricefloor.VolumeFactorCap {
		t.Errorf("volume factor should be capped at %.1f, got %.2f",
			pricefloor.VolumeFactorCap, result.VolumeFactor)
	}
}

func TestCleanMultiplierRange(t *testing.T) {
	// Fully clean grid (≤50 gCO2/kWh)
	if m := pricefloor.Multiplier(50); m != 1.5 {
		t.Errorf("clean grid multiplier: want 1.5, got %.2f", m)
	}
	// Fully dirty grid (≥700 gCO2/kWh)
	if m := pricefloor.Multiplier(700); m != 1.0 {
		t.Errorf("dirty grid multiplier: want 1.0, got %.2f", m)
	}
	// Midpoint
	m := pricefloor.Multiplier(375)
	if m < 1.0 || m > 1.5 {
		t.Errorf("midpoint multiplier out of range: %.2f", m)
	}
}

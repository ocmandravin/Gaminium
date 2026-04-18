package pricefloor

import (
	"fmt"
	"sync"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/ai"
)

// Calculator is the top-level price floor engine.
// It orchestrates energy fetching, carbon intensity data, AI validation,
// and floor formula calculation. Pure energy basis — no commodities.
type Calculator struct {
	energy    *EnergyIndex
	carbon    *CarbonFetcher
	validator *ai.Validator

	mu                   sync.RWMutex
	lastResult           *FloorResult
	lastUpdated          time.Time
	lastBlock            int64
	circulatingMinium    int64
	dailyVolumeMinium    int64
	baselineVolumeMinium int64
}

// NewCalculator creates a fully initialised price floor calculator.
func NewCalculator(validator *ai.Validator) *Calculator {
	return &Calculator{
		energy:               NewEnergyIndex(),
		carbon:               NewCarbonFetcher(),
		validator:            validator,
		circulatingMinium:    config.GenesisReward,
		baselineVolumeMinium: config.MiniumPerGMN * 1000, // 1000 GMN baseline
	}
}

// Update recomputes the price floor. Called every 2016 blocks by the node.
func (c *Calculator) Update(blockHeight int64, minerZone string) (*FloorResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Fetch energy data
	energyUSD, err := c.energy.Median()
	if err != nil {
		energyUSD = c.energy.RollingAverage()
		if energyUSD == 0 {
			energyUSD = 0.12 // global average fallback
		}
	}

	// 2. AI validate energy data
	energyDP := ai.DataPoint{
		Source:    "energy_index",
		Value:     energyUSD,
		Timestamp: time.Now(),
		Features:  []float64{energyUSD},
	}
	energyResult := c.validator.Validate(energyDP)
	if energyResult.Score.Rejected {
		return nil, fmt.Errorf("price floor: energy data rejected by AI validator: %s", energyResult.Reason)
	}
	energyUSD *= energyResult.Score.Weight

	// 3. Fetch clean energy multiplier for miner zone
	carbonData, _ := c.carbon.FetchCarbonIntensity(minerZone)
	cleanMult := CleanMultiplierMin
	if carbonData != nil {
		cleanMult = Multiplier(carbonData.GCO2kWh)
	}

	// 4. Compute floor
	inputs := FloorInputs{
		EnergyUSDkWh:           energyUSD,
		CleanEnergyMultiplier:  cleanMult,
		TotalSupplyMinium:      config.TotalSupplyMinum,
		CirculatingMinium:      c.circulatingMinium,
		DailyTxVolumeMinium:    c.dailyVolumeMinium,
		BaselineTxVolumeMinium: c.baselineVolumeMinium,
	}

	result := Calculate(inputs)
	c.lastResult = &result
	c.lastUpdated = time.Now()
	c.lastBlock = blockHeight

	return &result, nil
}

// CurrentFloor returns the last computed floor result without re-fetching.
func (c *Calculator) CurrentFloor() (*FloorResult, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastResult, c.lastUpdated
}

// UpdateChainState updates the circulating supply and volume from blockchain state.
func (c *Calculator) UpdateChainState(circulatingMinium, dailyVolumeMinium int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.circulatingMinium = circulatingMinium

	// EMA with 30-day window
	if c.baselineVolumeMinium == 0 {
		c.baselineVolumeMinium = dailyVolumeMinium
	} else {
		alpha := 1.0 / 30.0
		c.baselineVolumeMinium = int64(
			alpha*float64(dailyVolumeMinium) + (1-alpha)*float64(c.baselineVolumeMinium),
		)
	}
	c.dailyVolumeMinium = dailyVolumeMinium
}

// IsFloorBreached returns true if market price is within 2% of the floor.
func (c *Calculator) IsFloorBreached(marketPriceUSD float64) bool {
	floor, _ := c.CurrentFloor()
	if floor == nil || floor.Floor == 0 {
		return false
	}
	threshold := floor.Floor * (1.0 + config.FloorDefenseThreshold)
	return marketPriceUSD <= threshold
}

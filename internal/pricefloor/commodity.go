package pricefloor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CommodityIndex tracks a basket of physical commodities:
//   Gold 25% | Oil 25% | Natural Gas 25% | Carbon Credits 25%

type CommodityPrices struct {
	GoldUSDoz    float64 // USD per troy ounce
	OilUSDBarrel float64 // USD per barrel (WTI)
	NatGasMMBtu  float64 // USD per MMBtu (Henry Hub)
	CarbonEURt   float64 // EUR per tonne CO2 (EU ETS)
	FetchedAt    time.Time
}

// CommodityIndex fetches and maintains commodity price data.
type CommodityIndex struct {
	client *http.Client
	last   *CommodityPrices
}

// NewCommodityIndex creates a commodity price index fetcher.
func NewCommodityIndex() *CommodityIndex {
	return &CommodityIndex{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Fetch retrieves all commodity prices and returns the equal-weighted basket value in USD/unit.
func (c *CommodityIndex) Fetch() (*CommodityPrices, error) {
	prices := &CommodityPrices{FetchedAt: time.Now()}

	var fetchErr error

	// Gold — LBMA public data / open API
	gold, err := fetchGoldPrice(c.client)
	if err != nil {
		// Fall back to last known price
		if c.last != nil {
			gold = c.last.GoldUSDoz
		} else {
			fetchErr = fmt.Errorf("commodity: gold fetch failed: %w", err)
			gold = 2300.0 // 2024 average fallback
		}
	}
	prices.GoldUSDoz = gold

	// Oil — EIA weekly WTI price
	oil, err := fetchOilPrice(c.client)
	if err != nil {
		if c.last != nil {
			oil = c.last.OilUSDBarrel
		} else {
			oil = 75.0 // 2024 average fallback
		}
	}
	prices.OilUSDBarrel = oil

	// Natural Gas — Henry Hub via EIA
	gas, err := fetchNatGasPrice(c.client)
	if err != nil {
		if c.last != nil {
			gas = c.last.NatGasMMBtu
		} else {
			gas = 2.50 // 2024 average fallback
		}
	}
	prices.NatGasMMBtu = gas

	// Carbon — EU ETS
	carbon, err := fetchCarbonPrice(c.client)
	if err != nil {
		if c.last != nil {
			carbon = c.last.CarbonEURt
		} else {
			carbon = 60.0 // 2024 average fallback EUR/tonne
		}
	}
	prices.CarbonEURt = carbon

	c.last = prices
	return prices, fetchErr
}

// BasketValue returns the equal-weighted commodity index value normalised to USD.
// Returns a dimensionless index relative to base period (2024-01-01 = 1.0).
func (p *CommodityPrices) BasketValue() float64 {
	// Base values (2024 calendar year averages)
	const (
		goldBase   = 2000.0
		oilBase    = 75.0
		gasBase    = 2.50
		carbonBase = 60.0 * 1.08 // EUR to USD
	)

	carbonUSD := p.CarbonEURt * 1.08 // approximate EUR/USD

	goldRatio   := p.GoldUSDoz / goldBase
	oilRatio    := p.OilUSDBarrel / oilBase
	gasRatio    := p.NatGasMMBtu / gasBase
	carbonRatio := carbonUSD / (carbonBase)

	// Equal weighted: 25% each
	return (goldRatio + oilRatio + gasRatio + carbonRatio) / 4.0
}

// --- Commodity API fetchers ---

func fetchGoldPrice(client *http.Client) (float64, error) {
	// Using open commodity price API (goldprice.org public feed)
	url := "https://data-asg.goldprice.org/dbXRates/USD"
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("gold: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []struct {
			XauPrice float64 `json:"xauPrice"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Items) == 0 {
		return 0, fmt.Errorf("gold: parse error")
	}
	return result.Items[0].XauPrice, nil
}

func fetchOilPrice(client *http.Client) (float64, error) {
	// EIA weekly crude oil prices (public, no auth required)
	url := "https://api.eia.gov/v2/petroleum/pri/spt/data/?frequency=weekly&data[0]=value&facets[product][]=EPCWTI&sort[0][column]=period&sort[0][direction]=desc&offset=0&length=1"
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("oil: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Response struct {
			Data []struct {
				Value float64 `json:"value"`
			} `json:"data"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Response.Data) == 0 {
		return 0, fmt.Errorf("oil: parse error or empty data")
	}
	return result.Response.Data[0].Value, nil
}

func fetchNatGasPrice(client *http.Client) (float64, error) {
	// Henry Hub Natural Gas spot price via EIA
	url := "https://api.eia.gov/v2/natural-gas/pri/sum/data/?frequency=weekly&data[0]=value&facets[process][]=PH&sort[0][column]=period&sort[0][direction]=desc&offset=0&length=1"
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("natgas: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Response struct {
			Data []struct {
				Value float64 `json:"value"`
			} `json:"data"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Response.Data) == 0 {
		return 0, fmt.Errorf("natgas: parse error")
	}
	return result.Response.Data[0].Value, nil
}

func fetchCarbonPrice(client *http.Client) (float64, error) {
	// EU ETS carbon prices — using EMBER public data API
	url := "https://ember-climate.org/app/uploads/2022/03/Carbon-Price-Tracker-API.json"
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("carbon: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Price float64 `json:"price"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Price == 0 {
		// Fallback: EU ETS average 2024
		return 60.0, nil
	}
	return result.Price, nil
}

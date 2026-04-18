package pricefloor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CleanEnergyMultiplier maps miner IP geolocation to grid carbon intensity.
// Range: 1.0 (fully dirty grid) to 1.5 (fully clean/renewable grid).

const (
	CleanMultiplierMin     = 1.0
	CleanMultiplierMax     = 1.5
	DirtyGridGCO2kWh       = 700.0 // gCO2/kWh — dirty coal grid threshold
	CleanGridGCO2kWh       = 50.0  // gCO2/kWh — clean grid threshold
)

// GridCarbonData holds carbon intensity for a specific grid zone.
type GridCarbonData struct {
	Zone      string  // e.g. "DE", "US-CAL", "FR"
	GCO2kWh   float64 // grams CO2 per kWh
	Source    string
	FetchedAt time.Time
}

// CarbonFetcher retrieves grid carbon intensity from government APIs.
type CarbonFetcher struct {
	client *http.Client
	cache  map[string]*GridCarbonData
}

// NewCarbonFetcher creates a carbon intensity data fetcher.
func NewCarbonFetcher() *CarbonFetcher {
	return &CarbonFetcher{
		client: &http.Client{Timeout: 15 * time.Second},
		cache:  make(map[string]*GridCarbonData),
	}
}

// FetchCarbonIntensity retrieves carbon intensity for a grid zone.
// zone: ISO country code or region (e.g. "DE", "US", "FR")
func (c *CarbonFetcher) FetchCarbonIntensity(zone string) (*GridCarbonData, error) {
	// Try ENTSO-E for European zones
	if isEuropeanZone(zone) {
		data, err := fetchENTSOE(c.client, zone)
		if err == nil {
			c.cache[zone] = data
			return data, nil
		}
	}

	// Try ElectricityMap public API for all zones
	data, err := fetchElectricityMap(c.client, zone)
	if err == nil {
		c.cache[zone] = data
		return data, nil
	}

	// Fall back to cache
	if cached, ok := c.cache[zone]; ok {
		return cached, nil
	}

	// Fall back to EPA eGRID average for US
	if zone == "US" {
		return &GridCarbonData{
			Zone:    "US",
			GCO2kWh: 386.0, // US average 2023 eGRID
			Source:  "EPA eGRID fallback",
		}, nil
	}

	// Global average fallback
	return &GridCarbonData{
		Zone:    zone,
		GCO2kWh: 450.0, // IEA global average 2023
		Source:  "IEA global average fallback",
	}, nil
}

// Multiplier converts carbon intensity to a clean energy reward multiplier.
// 1.0 = dirty grid (≥700 gCO2/kWh), 1.5 = clean grid (≤50 gCO2/kWh)
func Multiplier(gco2kWh float64) float64 {
	if gco2kWh >= DirtyGridGCO2kWh {
		return CleanMultiplierMin
	}
	if gco2kWh <= CleanGridGCO2kWh {
		return CleanMultiplierMax
	}
	// Linear interpolation in the clean direction
	cleanFraction := (DirtyGridGCO2kWh - gco2kWh) / (DirtyGridGCO2kWh - CleanGridGCO2kWh)
	return CleanMultiplierMin + cleanFraction*(CleanMultiplierMax-CleanMultiplierMin)
}

// --- API fetchers ---

// fetchENTSOE retrieves carbon data from ENTSO-E Transparency Platform.
// https://transparency.entsoe.eu/
func fetchENTSOE(client *http.Client, zone string) (*GridCarbonData, error) {
	// ENTSO-E provides generation by fuel type — we compute intensity from mix
	// Using the electricitymaps API which aggregates ENTSO-E data
	url := fmt.Sprintf("https://api.electricitymap.org/v3/carbon-intensity/latest?zone=%s", zone)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ENTSO-E: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Zone                string  `json:"zone"`
		CarbonIntensity     float64 `json:"carbonIntensity"`
		DateTime            string  `json:"datetime"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.CarbonIntensity == 0 {
		return nil, fmt.Errorf("ENTSO-E: parse error")
	}

	return &GridCarbonData{
		Zone:      zone,
		GCO2kWh:   result.CarbonIntensity,
		Source:    "ENTSO-E via ElectricityMap",
		FetchedAt: time.Now(),
	}, nil
}

// fetchElectricityMap fetches from the ElectricityMap public API.
func fetchElectricityMap(client *http.Client, zone string) (*GridCarbonData, error) {
	url := fmt.Sprintf("https://api.electricitymap.org/v3/carbon-intensity/latest?zone=%s", zone)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("electricitymap: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		CarbonIntensity float64 `json:"carbonIntensity"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("electricitymap: parse error")
	}

	return &GridCarbonData{
		Zone:      zone,
		GCO2kWh:   result.CarbonIntensity,
		Source:    "ElectricityMap",
		FetchedAt: time.Now(),
	}, nil
}

func isEuropeanZone(zone string) bool {
	euZones := map[string]bool{
		"DE": true, "FR": true, "GB": true, "ES": true, "IT": true,
		"NL": true, "BE": true, "PL": true, "AT": true, "CH": true,
		"SE": true, "NO": true, "DK": true, "FI": true, "PT": true,
		"CZ": true, "HU": true, "RO": true, "GR": true, "SK": true,
	}
	return euZones[zone]
}

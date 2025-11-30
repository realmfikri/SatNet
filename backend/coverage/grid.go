package coverage

import (
	"errors"
	"math"
)

// EarthRadiusKm is the mean Earth radius in kilometers.
const EarthRadiusKm = 6371.0

// GridConfig controls the sampling resolution for coverage aggregation.
type GridConfig struct {
	LatStep float64 // degrees between latitude samples
	LonStep float64 // degrees between longitude samples
}

// Validate ensures the configuration is usable for generating a grid.
func (c GridConfig) Validate() error {
	if c.LatStep <= 0 || c.LonStep <= 0 {
		return errors.New("grid steps must be positive")
	}
	if c.LatStep > 180 || c.LonStep > 360 {
		return errors.New("grid steps are too large to tile the globe")
	}
	return nil
}

// Footprint represents the portion of Earth a satellite can service at an instant.
type Footprint struct {
	CenterLat    float64 // degrees
	CenterLon    float64 // degrees
	RadiusKm     float64 // kilometers
	LinkStrength float64 // arbitrary unit; larger indicates better link margin
}

// Cell captures aggregated coverage metrics for a single grid point.
type Cell struct {
	Lat           float64 // degrees
	Lon           float64 // degrees
	CoverageCount int
	StrongestLink float64
}

// Covered reports whether the cell is serviced by at least one footprint.
func (c Cell) Covered() bool {
	return c.CoverageCount > 0
}

// CoverageGrid holds the generated cells and supports aggregation of footprints.
type CoverageGrid struct {
	Config GridConfig
	cells  []Cell
}

// NewCoverageGrid builds a globe-spanning grid with the provided resolution.
// Cells are centered halfway into each step, beginning at -90/-180 degrees.
func NewCoverageGrid(config GridConfig) (*CoverageGrid, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var cells []Cell
	for lat := -90.0 + config.LatStep/2; lat < 90.0; lat += config.LatStep {
		for lon := -180.0 + config.LonStep/2; lon < 180.0; lon += config.LonStep {
			cells = append(cells, Cell{Lat: lat, Lon: lon})
		}
	}

	return &CoverageGrid{Config: config, cells: cells}, nil
}

// ApplyFootprints increments coverage metrics for cells inside the provided footprints.
func (g *CoverageGrid) ApplyFootprints(footprints []Footprint) {
	for i := range g.cells {
		cell := &g.cells[i]
		for _, footprint := range footprints {
			if footprint.RadiusKm <= 0 {
				continue
			}
			if pointInsideFootprint(cell.Lat, cell.Lon, footprint) {
				cell.CoverageCount++
				if footprint.LinkStrength > cell.StrongestLink {
					cell.StrongestLink = footprint.LinkStrength
				}
			}
		}
	}
}

// Summary captures high-level visibility statistics for the grid.
type Summary struct {
	TotalCells       int
	CoveredCells     int
	CoveragePercent  float64
	UncoveredSamples []GapSample
}

// GapSample represents a gap in coverage suitable for surfacing on a heatmap.
type GapSample struct {
	Lat float64
	Lon float64
}

// Summarize returns coverage statistics and gap locations.
func (g *CoverageGrid) Summarize() Summary {
	var covered int
	var gaps []GapSample

	for _, cell := range g.cells {
		if cell.Covered() {
			covered++
		} else {
			gaps = append(gaps, GapSample{Lat: cell.Lat, Lon: cell.Lon})
		}
	}

	total := len(g.cells)
	percent := 0.0
	if total > 0 {
		percent = (float64(covered) / float64(total)) * 100.0
	}

	return Summary{
		TotalCells:       total,
		CoveredCells:     covered,
		CoveragePercent:  percent,
		UncoveredSamples: gaps,
	}
}

// HeatmapCell is a frontend-friendly payload describing a cell's coverage strength.
type HeatmapCell struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Covered  bool    `json:"covered"`
	Count    int     `json:"count"`
	Strength float64 `json:"strength"`
}

// HeatmapData exports coverage information formatted for the UI heatmap.
func (g *CoverageGrid) HeatmapData() []HeatmapCell {
	heatmap := make([]HeatmapCell, 0, len(g.cells))
	for _, cell := range g.cells {
		heatmap = append(heatmap, HeatmapCell{
			Lat:      cell.Lat,
			Lon:      cell.Lon,
			Covered:  cell.Covered(),
			Count:    cell.CoverageCount,
			Strength: cell.StrongestLink,
		})
	}
	return heatmap
}

func pointInsideFootprint(lat, lon float64, footprint Footprint) bool {
	distance := haversineDistanceKm(lat, lon, footprint.CenterLat, footprint.CenterLon)
	return distance <= footprint.RadiusKm
}

func haversineDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const degToRad = math.Pi / 180
	dLat := (lat2 - lat1) * degToRad
	dLon := (lon2 - lon1) * degToRad

	lat1Rad := lat1 * degToRad
	lat2Rad := lat2 * degToRad

	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)

	a := sinLat*sinLat + sinLon*sinLon*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// Cells exposes a copy of the grid cells to callers that need to inspect raw results.
func (g *CoverageGrid) Cells() []Cell {
	out := make([]Cell, len(g.cells))
	copy(out, g.cells)
	return out
}

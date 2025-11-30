package coverage

import (
	"math"
	"testing"
)

func TestNewCoverageGrid(t *testing.T) {
	grid, err := NewCoverageGrid(GridConfig{LatStep: 20, LonStep: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cells := grid.Cells()
	expected := 81 // 9 latitude bands * 9 longitude bands
	if len(cells) != expected {
		t.Fatalf("expected %d cells, got %d", expected, len(cells))
	}

	first := cells[0]
	if math.Abs(first.Lat-(-80)) > 1e-9 || math.Abs(first.Lon-(-160)) > 1e-9 {
		t.Fatalf("unexpected first cell center: %+v", first)
	}
}

func TestApplyFootprintsAndSummarize(t *testing.T) {
	grid, err := NewCoverageGrid(GridConfig{LatStep: 20, LonStep: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	footprints := []Footprint{
		{CenterLat: 0, CenterLon: 0, RadiusKm: 1200, LinkStrength: 5},
		{CenterLat: 0, CenterLon: 0, RadiusKm: 1300, LinkStrength: 7},
		{CenterLat: 40, CenterLon: 40, RadiusKm: 800, LinkStrength: 12},
	}

	grid.ApplyFootprints(footprints)

	cells := grid.Cells()
	var equator Cell
	var inclined Cell

	for _, cell := range cells {
		if math.Abs(cell.Lat-0) < 1e-9 && math.Abs(cell.Lon-0) < 1e-9 {
			equator = cell
		}
		if math.Abs(cell.Lat-40) < 1e-9 && math.Abs(cell.Lon-40) < 1e-9 {
			inclined = cell
		}
	}

	if equator.CoverageCount != 2 {
		t.Fatalf("expected equator cell to be hit twice, got %d", equator.CoverageCount)
	}
	if equator.StrongestLink != 7 {
		t.Fatalf("expected strongest link 7 at equator, got %f", equator.StrongestLink)
	}
	if inclined.CoverageCount != 1 || inclined.StrongestLink != 12 {
		t.Fatalf("unexpected inclined metrics: %+v", inclined)
	}

	summary := grid.Summarize()
	if summary.TotalCells != 81 {
		t.Fatalf("expected 81 total cells, got %d", summary.TotalCells)
	}
	if summary.CoveredCells != 2 {
		t.Fatalf("expected 2 covered cells, got %d", summary.CoveredCells)
	}
	expectedPercent := (2.0 / 81.0) * 100
	if math.Abs(summary.CoveragePercent-expectedPercent) > 1e-9 {
		t.Fatalf("expected coverage percent %f, got %f", expectedPercent, summary.CoveragePercent)
	}
	if len(summary.UncoveredSamples) != 79 {
		t.Fatalf("expected 79 uncovered samples, got %d", len(summary.UncoveredSamples))
	}
}

func TestHeatmapData(t *testing.T) {
	grid, err := NewCoverageGrid(GridConfig{LatStep: 30, LonStep: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	grid.ApplyFootprints([]Footprint{{CenterLat: -45, CenterLon: -150, RadiusKm: 1500, LinkStrength: 9}})

	heatmap := grid.HeatmapData()
	if len(heatmap) != 36 {
		t.Fatalf("expected 36 heatmap entries, got %d", len(heatmap))
	}

	var covered HeatmapCell
	var found bool
	for _, cell := range heatmap {
		if math.Abs(cell.Lat-(-45)) < 1e-9 && math.Abs(cell.Lon-(-150)) < 1e-9 {
			covered = cell
			found = true
		}
	}

	if !found {
		t.Fatalf("expected to find covered cell in heatmap data")
	}
	if !covered.Covered || covered.Count != 1 || covered.Strength != 9 {
		t.Fatalf("unexpected heatmap metrics: %+v", covered)
	}
}

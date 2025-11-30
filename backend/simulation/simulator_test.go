package simulation

import (
	"testing"
	"time"

	"github.com/example/satnet/backend/coverage"
	"github.com/example/satnet/backend/visibility"
)

func TestTrafficReroutesAfterSatelliteFailure(t *testing.T) {
	cfg := Config{
		GridConfig:    coverage.GridConfig{LatStep: 180, LonStep: 360},
		ElevationMask: 0,
		Satellites: []Satellite{
			{ID: "primary", Position: visibility.Vector3{X: visibility.EarthRadius + 300, Y: 0, Z: 0}, Footprint: coverage.Footprint{CenterLat: 0, CenterLon: 0, RadiusKm: 1200, LinkStrength: 1}},
			{ID: "backup", Position: visibility.Vector3{X: visibility.EarthRadius + 900, Y: 200, Z: 0}, Footprint: coverage.Footprint{CenterLat: 70, CenterLon: 90, RadiusKm: 400, LinkStrength: 0.5}},
		},
		GroundStations: []GroundStation{
			{ID: "ground-a", Position: visibility.Vector3{X: visibility.EarthRadius, Y: 0, Z: 0}},
			{ID: "ground-b", Position: visibility.Vector3{X: visibility.EarthRadius, Y: 20, Z: 0}},
		},
		Traffic: []TrafficDemand{{ID: "g1-to-g2", FromID: "ground-a", ToID: "ground-b"}},
	}

	sim, err := NewSimulator(cfg)
	if err != nil {
		t.Fatalf("failed to build simulator: %v", err)
	}
	drainEvents(sim)

	initial := sim.Snapshot()
	initialPath, ok := initial.Routes["g1-to-g2"]
	if !ok {
		t.Fatalf("expected initial route for demand")
	}
	if len(initialPath.Nodes) < 2 || initialPath.Nodes[1] != "primary" {
		t.Fatalf("expected primary satellite in initial route, got %v", initialPath.Nodes)
	}

	updated, err := sim.DisableSatellite("primary")
	if err != nil {
		t.Fatalf("disable failed: %v", err)
	}

	rerouted, ok := updated.Routes["g1-to-g2"]
	if !ok {
		t.Fatalf("expected rerouted path")
	}
	if contains(rerouted.Nodes, "primary") {
		t.Fatalf("route should not include disabled satellite: %v", rerouted.Nodes)
	}
	if len(rerouted.Nodes) < 2 || rerouted.Nodes[1] != "backup" {
		t.Fatalf("expected backup satellite in rerouted path, got %v", rerouted.Nodes)
	}

	event := waitForEvent(t, sim, EventTopologyUpdated)
	if event.Snapshot.Timestamp.Before(updated.Timestamp) {
		t.Fatalf("expected event snapshot to reflect latest recompute")
	}
}

func TestCoverageUpdatesAfterRemoval(t *testing.T) {
	cfg := Config{
		GridConfig:    coverage.GridConfig{LatStep: 180, LonStep: 360},
		ElevationMask: 0,
		Satellites: []Satellite{
			{ID: "covering", Position: visibility.Vector3{X: visibility.EarthRadius + 400, Y: 0, Z: 0}, Footprint: coverage.Footprint{CenterLat: 0, CenterLon: 0, RadiusKm: 1500, LinkStrength: 1}},
			{ID: "far-away", Position: visibility.Vector3{X: visibility.EarthRadius + 900, Y: 800, Z: 0}, Footprint: coverage.Footprint{CenterLat: 70, CenterLon: 120, RadiusKm: 200, LinkStrength: 0.2}},
		},
		GroundStations: []GroundStation{{ID: "ground", Position: visibility.Vector3{X: visibility.EarthRadius, Y: 0, Z: 0}}},
		Traffic:        []TrafficDemand{{ID: "ping", FromID: "ground", ToID: "ground"}},
	}

	sim, err := NewSimulator(cfg)
	if err != nil {
		t.Fatalf("failed to build simulator: %v", err)
	}
	drainEvents(sim)

	baseline := sim.Snapshot()
	if baseline.Coverage.CoveragePercent != 100 {
		t.Fatalf("expected full coverage with covering satellite, got %.2f", baseline.Coverage.CoveragePercent)
	}

	updated, err := sim.RemoveSatellite("covering")
	if err != nil {
		t.Fatalf("removal failed: %v", err)
	}
	if updated.Coverage.CoveragePercent != 0 {
		t.Fatalf("expected coverage to drop after removal, got %.2f", updated.Coverage.CoveragePercent)
	}

	event := waitForEvent(t, sim, EventCoverageUpdated)
	if event.Snapshot.Coverage.CoveragePercent != updated.Coverage.CoveragePercent {
		t.Fatalf("event payload should reflect recomputed coverage; got %.2f", event.Snapshot.Coverage.CoveragePercent)
	}
}

func drainEvents(sim *Simulator) {
	for {
		select {
		case <-sim.Events():
			continue
		default:
			return
		}
	}
}

func waitForEvent(t *testing.T, sim *Simulator, eventType EventType) Event {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case evt := <-sim.Events():
			if evt.Type == eventType {
				return evt
			}
		case <-timeout:
			t.Fatalf("timed out waiting for event type %s", eventType)
		}
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
